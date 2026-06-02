package crypto

import (
	"crypto/ed25519"
	"testing"
)

func TestGenerateKeyPair_Lengths(t *testing.T) {
	seed, pub, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(seed) != 32 {
		t.Errorf("seed length: got %d, want 32", len(seed))
	}
	if len(pub) != 32 {
		t.Errorf("pubKey length: got %d, want 32", len(pub))
	}
}

func TestSignAndVerify_Valid(t *testing.T) {
	seed, pub, _ := GenerateKeyPair()
	msg := []byte("test message for chat")

	// Восстанавливаем полный приватный ключ из seed для подписи (клиентская часть)
	priv := ed25519.NewKeyFromSeed(seed)
	sig := ed25519.Sign(priv, msg)

	if !Verify(pub, sig, msg) {
		t.Error("valid signature rejected")
	}
}

func TestVerify_TamperedMessage(t *testing.T) {
	seed, pub, _ := GenerateKeyPair()
	priv := ed25519.NewKeyFromSeed(seed)
	msg := []byte("original")
	sig := ed25519.Sign(priv, msg)

	if Verify(pub, sig, []byte("tampered")) {
		t.Error("tampered message should fail verification")
	}
}

func TestVerify_TamperedSignature(t *testing.T) {
	seed, pub, _ := GenerateKeyPair()
	priv := ed25519.NewKeyFromSeed(seed)
	msg := []byte("message")
	sig := ed25519.Sign(priv, msg)

	tampered := make([]byte, len(sig))
	copy(tampered, sig)
	tampered[0] ^= 0xFF

	if Verify(pub, tampered, msg) {
		t.Error("tampered signature should fail verification")
	}
}

func TestDecodeBase64_InvalidInput(t *testing.T) {
	cases := []string{
		"not-valid-base64!!!",
		"SGVsbG8=",  // valid
		"SGVsbG8",   // missing padding (std decoder rejects)
		"SGVsbG8!=", // illegal character
	}

	for _, tc := range cases {
		t.Run(tc, func(t *testing.T) {
			_, err := DecodeBase64(tc)
			// Ожидаем ошибку для битых строк
			if tc == "SGVsbG8=" {
				if err != nil {
					t.Errorf("expected success for valid base64, got: %v", err)
				}
				return
			}
			if err == nil {
				t.Errorf("expected error for invalid base64 %q", tc)
			}
		})
	}
}

func TestDecodeBase64_WrongLengths(t *testing.T) {
	_, err := DecodeBase64("AQID") // 3 байта
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// В service/api мы будем дополнительно проверять len(b) == 32/64,
	// но сам декодер не должен паниковать
}

func TestVerify_WrongKeyOrSigLengths(t *testing.T) {
	_, pub, _ := GenerateKeyPair()

	tests := []struct {
		name     string
		pubKey   []byte
		sig      []byte
		message  []byte
		wantTrue bool
	}{
		{"truncated pubkey", pub[:15], make([]byte, 64), []byte("msg"), false},
		{"truncated sig", pub, make([]byte, 63), []byte("msg"), false},
		{"empty pubkey", []byte{}, make([]byte, 64), []byte("msg"), false},
		{"empty sig", pub, []byte{}, []byte("msg"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Verify(tt.pubKey, tt.sig, tt.message)
			if got != tt.wantTrue {
				t.Errorf("Verify() = %v, want %v", got, tt.wantTrue)
			}
		})
	}
}
