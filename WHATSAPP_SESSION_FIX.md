# WhatsApp Session & Encryption Fix

## Masalah yang Diperbaiki

### 1. Session WhatsApp Tidak Terbaca
**Problem**: Meskipun file `whatsmeow.db` sudah ada, aplikasi masih meminta login ulang.

**Root Cause**: 
- Menggunakan `container.NewDevice()` yang selalu membuat device baru
- Tidak menggunakan device yang sudah tersimpan di database

**Solution**:
```go
// Sebelum (SALAH)
device := container.NewDevice()

// Sesudah (BENAR)
deviceStore, err := container.GetFirstDevice(context.Background())
if err != nil {
    log.Printf("No existing device found, creating new device: %v", err)
    deviceStore = container.NewDevice()
} else {
    log.Printf("Found existing device with ID: %s", deviceStore.ID)
}
```

### 2. Error Enkripsi WhatsApp
**Problem**: 
```
[Client WARN] Failed to encrypt 3EB05C2EC963BD4F5A0F48 for 6285704384348@s.whatsapp.net: 
can't encrypt message for device: no signal session established with 266670375469161_1:0
```

**Root Cause**:
- Session belum sepenuhnya established
- Tidak ada retry mechanism untuk encryption errors
- Koneksi tidak stabil saat startup

**Solution**:
1. **Connection Stabilization**:
   ```go
   // Wait for connection to stabilize
   log.Println("Waiting for connection to stabilize...")
   time.Sleep(3 * time.Second)
   ```

2. **Event Monitoring**:
   ```go
   client.AddEventHandler(func(evt interface{}) {
       switch v := evt.(type) {
       case *waEvents.Connected:
           log.Println("WhatsApp client connected successfully")
       case *waEvents.Disconnected:
           log.Printf("WhatsApp client disconnected: %v", v.Reason)
       }
   })
   ```

3. **Retry Mechanism**:
   ```go
   maxRetries := 3
   for i := 0; i < maxRetries; i++ {
       resp, err = w.client.SendMessage(ctx, to, msg)
       if err == nil {
           break
       }
       
       if strings.Contains(fmt.Sprintf("%v", err), "can't encrypt message") {
           log.Printf("[WA] Encryption error (attempt %d/%d): %v", i+1, maxRetries, err)
           time.Sleep(time.Duration(i+1) * 2 * time.Second)
           continue
       }
       break
   }
   ```

## Fitur Tambahan

### Auto-Revoke Message
Pesan akan otomatis di-revoke/unsend setelah 1 menit:
```go
go func(messageID string, jid waTypes.JID) {
    select {
    case <-time.After(1 * time.Minute):
        revoke := w.client.BuildRevoke(jid, w.client.Store.ID.ToNonAD(), messageID)
        w.client.SendMessage(context.Background(), jid, revoke)
    }
}(resp.ID, to)
```

## Testing

1. **Build**: `go build ./cmd/bot`
2. **Run**: `./bot.exe`
3. **Test API**: 
   ```bash
   curl -X POST http://localhost:8080/api/send-message \
     -H "Content-Type: application/json" \
     -H "X-API-Key: your-api-key" \
     -d '{"phone": "6285704384348", "message": "Test message"}'
   ```

## Logs yang Diharapkan

```
Initializing WhatsApp service with store path: whatsmeow.db
Found existing device with ID: [device-id]
Existing session found for device ID: [device-id]
Waiting for connection to stabilize...
WhatsApp client connected successfully
Successfully connected using existing session!
```

## Troubleshooting

1. **Jika masih login ulang**: Hapus `whatsmeow.db` dan scan QR code sekali lagi
2. **Jika encryption error**: Tunggu beberapa detik, sistem akan retry otomatis
3. **Jika connection timeout**: Periksa koneksi internet dan firewall