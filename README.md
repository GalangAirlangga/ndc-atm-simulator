# NDC ATM Simulator

Simulator sederhana untuk mengirim pesan NDC (NCR Direct Connection) ke host/ATM melalui TCP. Pesan bisa dibuat manual atau menggunakan builder NDC klasik (Class + Type + Sub-type + LUNO).

## Requirements

- Go 1.21+
- Python 3.6+ (untuk mock server)

## Build

```bash
go build -o ndcatm .
```

## Penggunaan Cepat

### Kirim pesan hex satu kali

```bash
./ndcatm -host 192.168.1.10:3000 -hex '3132330a'
```

### Kirim pesan NDC klasik

```bash
./ndcatm -host 192.168.1.10:3000 \
  -class 1 -type 11 -subtype 1 -luno 001 \
  -hex '3132330a'
```

### Mode interaktif (koneksi tetap terbuka)

```bash
./ndcatm -host 192.168.1.10:3000 -i
```

Di dalam mode interaktif, ketik:

```
ndc> hex 3132330a
ndc> ascii Hello
ndc> ndc 1 11 1 001 3132330a
ndc> ndc-ascii 2 22 1 001 hello world
ndc> prefix uint16be
ndc> quit
```

## Opsi Baris Perintah

| Flag | Contoh | Keterangan |
|------|--------|-----------|
| `-host` | `192.168.1.10:3000` | Target host:port (wajib) |
| `-hex` | `3132330a` | Pesan dalam format hex (spasi boleh) |
| `-ascii` | `Hello` | Pesan dalam format ASCII |
| `-class` | `1` | Message Class (1 digit) |
| `-type` | `22` | Message Type (2 digit) |
| `-subtype` | `1` | Message Sub-type (1 digit) |
| `-luno` | `001` | LUNO (3 digit) |
| `-prefix` | `uint16be` | Length prefix: `none` / `uint16be` / `ascii2` / `ascii4` |
| `-read` | `false` | Baca response setelah kirim (`true`/`false`) |
| `-timeout` | `5s` | Timeout koneksi/baca |
| `-i` | — | Mode interaktif (koneksi tetap terbuka) |

## Contoh Pesan NDC Klasik

Field separator (FS) pada NDC adalah byte `0x1C`. Untuk mengirimnya gunakan `-hex`.

### Message Class (kelas pesan)

| Kelas | Keterangan |
|-------|-----------|
| `1` | Terminal → Host (request) |
| `2` | Host → Terminal (reply/command) |
| `3` | Status message |

### Message Type & Sub-type populer

| Type | Sub-type | Keterangan |
|------|----------|-----------|
| `11` | `1` | Transaction Request (tarik tunai) |
| `40` | `0` | Transaction Reply (otorisasi) |
| `22` | `1` | Unsolicited Status (status perangkat) |
| `11` | `0` | Authentication Request |
| `21` | `0` | Authentication Reply |

### Contoh pesan lengkap

#### 1. Unsolicited Status (Terminal Ready)

```bash
./ndcatm -host 192.168.1.10:3000 \
  -hex '32321c3030311c3030301c391c42'
```

ASCII: `22<FS>001<FS>000<FS>9<FS>B`  
Keterangan: status device terminal ready.

#### 2. Transaction Request (tarik tunai)

```bash
./ndcatm -host 192.168.1.10:3000 \
  -hex '31311c3030311c3030301c3132331c3b343030303132333435363738393031303d3236313231303130303030303030303f1c3030301c3030303030313030303030301c31323334353637383930414243444546'
```

#### 3. Command: Go In-Service (dengan length prefix 2 byte)

```bash
./ndcatm -host 192.168.1.10:3000 \
  -prefix uint16be \
  -hex '31301c3030311c3030301c31'
```

Hex lengkap: `000a` (panjang 10) + `31301c3030311c3030301c31`

## Test Lokal dengan Mock Server

### Menggunakan script otomatis

```bash
./test_local.sh
```

Script ini akan:
1. Build simulator
2. Menjalankan mock server di port 12355
3. Mengirim pesan NDC example
4. Menampilkan hasil terima-kirim

### Jalankan manual

Terminal 1 — mock server:

```bash
python3 mock_server.py
```

Terminal 2 — kirim pesan:

```bash
./ndcatm -host 127.0.0.1:12355 \
  -class 1 -type 11 -subtype 1 -luno 001 \
  -ascii 'Hello'
```

## Structure

```
ndc-atm-simulator/
├── main.go          # Source Go
├── go.mod           # Go module
├── mock_server.py   # Mock server untuk test lokal
├── test_local.sh    # Script test otomatis
└── README.md        # Dokumentasi ini
```

## Catatan

- Semua data dikirim sebagai raw bytes ke TCP socket.
- Pesan NDC menggunakan karakter FS (`0x1C`) sebagai field separator.
- Mode interaktif menjaga koneksi tetap terbuka dan menampilkan pesan masuk secara real-time.
- `-prefix uint16be` menambahkan 2-byte length header di depan payload (umum dipakai NDC+).
