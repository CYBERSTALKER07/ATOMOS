import re

with open('pegasus/apps/backend-go/vault/vault.go', 'r') as f:
    content = f.read()

bad_block = """cfg.SecretKey, err = s.decrypt(encryptedKey)
                                if err == nil {
                                        return &cfg, nil
                                }"""

good_block = """decrypted, err := Decrypt(encryptedKey)
                                if err == nil {
                                        cfg.SecretKey = string(decrypted)
                                        return &cfg, nil
                                }"""

content = content.replace(bad_block, good_block)

with open('pegasus/apps/backend-go/vault/vault.go', 'w') as f:
    f.write(content)
