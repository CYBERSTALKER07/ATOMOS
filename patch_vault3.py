with open('pegasus/apps/backend-go/vault/vault.go', 'r') as f:
    content = f.read()
    
import re
content = re.sub(r'cfg\.SecretKey, err = s\.decrypt\(encryptedKey\)\s+if err == nil {\s+return &cfg, nil\s+}', r'''decrypted, err := Decrypt(encryptedKey)
                                if err == nil {
                                        cfg.SecretKey = string(decrypted)
                                        return &cfg, nil
                                }''', content)
with open('pegasus/apps/backend-go/vault/vault.go', 'w') as f:
    f.write(content)
