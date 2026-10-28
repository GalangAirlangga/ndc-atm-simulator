package main

import (
	"bufio"
	"encoding/hex"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

type prefixType string

const (
	prefixNone   prefixType = "none"
	prefixUint16 prefixType = "uint16be"
	prefixAscii2 prefixType = "ascii2"
	prefixAscii4 prefixType = "ascii4"
)

func main() {
	host := flag.String("host", "", "Target host:port, e.g. 192.168.1.10:3000")
	msgHex := flag.String("hex", "", "NDC message sebagai hex string (spasi boleh)")
	msgAscii := flag.String("ascii", "", "NDC message sebagai string ASCII")
	prefix := flag.String("prefix", "none", "Length prefix: none | uint16be | ascii2 | ascii4")
	timeout := flag.Duration("timeout", 5*time.Second, "TCP connect/read timeout")
	read := flag.Bool("read", true, "Baca response setelah kirim")
	interactive := flag.Bool("i", false, "Mode interaktif (pertahankan koneksi)")

	// NDC classic builder fields
	ndcClass := flag.String("class", "", "NDC Message Class (1 digit)")
	ndcType := flag.String("type", "", "NDC Message Type (2 digit)")
	ndcSubType := flag.String("subtype", "", "NDC Message Sub-type (1 digit)")
	ndcLuno := flag.String("luno", "", "NDC LUNO (3 digit)")

	flag.Parse()

	if *host == "" {
		fmt.Fprintln(os.Stderr, "Error: -host wajib diisi, contoh: -host 192.168.1.10:3000")
		flag.Usage()
		os.Exit(1)
	}

	prefixVal := prefixType(strings.ToLower(*prefix))
	if err := validatePrefix(prefixVal); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if *interactive {
		runInteractive(*host, prefixVal, *timeout)
		return
	}

	var payload []byte
	var err error

	if *ndcClass != "" || *ndcType != "" || *ndcSubType != "" || *ndcLuno != "" {
		payload, err = buildNDC(*ndcClass, *ndcType, *ndcSubType, *ndcLuno, *msgHex, *msgAscii)
		if err != nil {
			fmt.Fprintf(os.Stderr, "NDC build error: %v\n", err)
			os.Exit(1)
		}
	} else if *msgHex != "" {
		payload, err = parseHex(*msgHex)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Hex error: %v\n", err)
			os.Exit(1)
		}
	} else if *msgAscii != "" {
		payload = []byte(*msgAscii)
	} else {
		payload, err = readStdin()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Stdin error: %v\n", err)
			os.Exit(1)
		}
	}

	if len(payload) == 0 {
		fmt.Fprintln(os.Stderr, "Error: pesan kosong. Gunakan -hex, -ascii, -class/-type/-subtype/-luno, atau pipa stdin.")
		os.Exit(1)
	}

	data, err := wrapPrefix(payload, prefixVal)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Prefix error: %v\n", err)
		os.Exit(1)
	}

	if err := sendOnce(*host, data, *timeout, *read); err != nil {
		fmt.Fprintf(os.Stderr, "Send error: %v\n", err)
		os.Exit(1)
	}
}

func buildNDC(class, msgType, subType, luno, hexBody, asciiBody string) ([]byte, error) {
	if len(class) != 1 || len(msgType) != 2 || len(subType) != 1 || len(luno) != 3 {
		return nil, fmt.Errorf("format NDC harus: class(1) type(2) sub-type(1) LUNO(3)")
	}
	var body []byte
	var err error
	if hexBody != "" {
		body, err = parseHex(hexBody)
		if err != nil {
			return nil, fmt.Errorf("body hex error: %w", err)
		}
	} else {
		body = []byte(asciiBody)
	}
	header := class + msgType + subType + luno
	return append([]byte(header), body...), nil
}

func validatePrefix(p prefixType) error {
	switch p {
	case prefixNone, prefixUint16, prefixAscii2, prefixAscii4:
		return nil
	}
	return fmt.Errorf("prefix tidak dikenal: %s", p)
}

func parseHex(s string) ([]byte, error) {
	s = strings.ReplaceAll(s, " ", "")
	return hex.DecodeString(s)
}

func readStdin() ([]byte, error) {
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil && err.Error() != "EOF" {
		return nil, err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, nil
	}
	return parseHex(line)
}

func wrapPrefix(payload []byte, p prefixType) ([]byte, error) {
	n := len(payload)
	switch p {
	case prefixNone:
		return payload, nil
	case prefixUint16:
		if n > 0xFFFF {
			return nil, fmt.Errorf("payload terlalu besar untuk uint16be")
		}
		header := []byte{byte(n >> 8), byte(n)}
		return append(header, payload...), nil
	case prefixAscii2:
		if n > 99 {
			return nil, fmt.Errorf("payload terlalu besar untuk ascii2")
		}
		header := []byte(fmt.Sprintf("%02d", n))
		return append(header, payload...), nil
	case prefixAscii4:
		if n > 9999 {
			return nil, fmt.Errorf("payload terlalu besar untuk ascii4")
		}
		header := []byte(fmt.Sprintf("%04d", n))
		return append(header, payload...), nil
	}
	return nil, fmt.Errorf("prefix tidak dikenal")
}

func sendOnce(host string, data []byte, timeout time.Duration, readResponse bool) error {
	conn, err := net.DialTimeout("tcp", host, timeout)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return err
	}

	fmt.Printf("[connect] %s\n", host)
	fmt.Printf("[send] %d bytes: %s\n", len(data), hex.EncodeToString(data))

	if _, err := conn.Write(data); err != nil {
		return err
	}

	if !readResponse {
		return nil
	}

	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			fmt.Println("[recv] timeout (no response)")
			return nil
		}
		return err
	}
	fmt.Printf("[recv] %d bytes: %s\n", n, hex.EncodeToString(buf[:n]))
	printable := makePrintable(buf[:n])
	fmt.Printf("[recv] text: %s\n", printable)
	return nil
}

func runInteractive(host string, p prefixType, timeout time.Duration) {
	conn, err := net.DialTimeout("tcp", host, timeout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Connect error: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()
	fmt.Printf("[connect] %s\n", host)
	fmt.Println("Perintah: hex <hex> | ascii <text> | ndc <c> <t> <s> <luno> <body-hex> | ndc-ascii <c> <t> <s> <luno> <body> | prefix <...> | quit")
	fmt.Println("Incoming messages akan muncul otomatis.")

	// Goroutine ini terus membaca data dari server agar koneksi tetap terbuka
	// dan pesan yang dikirim server muncul kapan saja.
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := conn.Read(buf)
			if err != nil {
				// Koneksi ditutup atau error lain.
				return
			}
			if n > 0 {
				fmt.Printf("\n[recv] %d bytes: %s\n", n, hex.EncodeToString(buf[:n]))
				fmt.Printf("[recv] text: %s\n", makePrintable(buf[:n]))
				fmt.Print("ndc> ")
			}
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)
	currentPrefix := p

	for {
		fmt.Print("ndc> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		cmd := strings.ToLower(parts[0])
		arg := ""
		if len(parts) > 1 {
			arg = strings.TrimSpace(parts[1])
		}

		switch cmd {
		case "quit", "exit":
			fmt.Println("[disconnect]")
			time.Sleep(150 * time.Millisecond)
			return
		case "prefix":
			newP := prefixType(arg)
			if err := validatePrefix(newP); err != nil {
				fmt.Println(err)
				continue
			}
			currentPrefix = newP
			fmt.Printf("[prefix] %s\n", currentPrefix)
		case "read":
			fmt.Println("[read] command tidak dipakai lagi; incoming messages otomatis tampil")
		case "hex":
			payload, err := parseHex(arg)
			if err != nil {
				fmt.Printf("Hex error: %v\n", err)
				continue
			}
			if err := sendInteractive(conn, payload, currentPrefix); err != nil {
				fmt.Printf("Send error: %v\n", err)
			}
		case "ascii":
			if err := sendInteractive(conn, []byte(arg), currentPrefix); err != nil {
				fmt.Printf("Send error: %v\n", err)
			}
		case "ndc", "ndc-ascii":
			if err := handleInteractiveNDC(conn, arg, currentPrefix, cmd == "ndc-ascii"); err != nil {
				fmt.Printf("NDC error: %v\n", err)
			}
		default:
			fmt.Println("Perintah tidak dikenal.")
		}
	}
}

func handleInteractiveNDC(conn net.Conn, arg string, p prefixType, ascii bool) error {
	fields := strings.Fields(arg)
	if len(fields) < 5 {
		return fmt.Errorf("usage: ndc <class> <type> <sub-type> <luno> <body>")
	}
	var hexBody, asciiBody string
	if ascii {
		hexBody = ""
		asciiBody = strings.Join(fields[4:], " ")
	} else {
		hexBody = strings.Join(fields[4:], " ")
		asciiBody = ""
	}
	payload, err := buildNDC(fields[0], fields[1], fields[2], fields[3], hexBody, asciiBody)
	if err != nil {
		return err
	}
	return sendInteractive(conn, payload, p)
}

func sendInteractive(conn net.Conn, payload []byte, p prefixType) error {
	data, err := wrapPrefix(payload, p)
	if err != nil {
		return err
	}
	fmt.Printf("[send] %d bytes: %s\n", len(data), hex.EncodeToString(data))
	_, err = conn.Write(data)
	return err
}

func makePrintable(b []byte) string {
	s := string(b)
	var out strings.Builder
	for _, r := range s {
		if r >= 32 && r < 127 {
			out.WriteRune(r)
		} else {
			out.WriteRune('.')
		}
	}
	return out.String()
}
