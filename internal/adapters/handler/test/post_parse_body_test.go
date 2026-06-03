package handler_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/pavel/anonymous-chat/internal/adapters/handler"
)

// Проверка на все возможные граничные случаи и успешные при парсинге трех величин передающихся в POST

func TestParsePostBody(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       []byte
		wantPubKey  string
		wantSig     string
		wantContent []byte
		wantErr     bool
		errContains string // Опциональная проверка подстроки в ошибке
	}{
		// ✅ Валидные кейсы
		{
			name:        "simple message with LF",
			input:       []byte("pubkey sig hello world\n"),
			wantPubKey:  "pubkey",
			wantSig:     "sig",
			wantContent: []byte("hello world"),
			wantErr:     false,
		},
		{
			name:        "CRLF line ending (Windows)",
			input:       []byte("pubkey sig hello world\r\n"),
			wantPubKey:  "pubkey",
			wantSig:     "sig",
			wantContent: []byte("hello world"),
			wantErr:     false,
		},
		{
			name:        "multiple spaces between fields",
			input:       []byte("pubkey  sig   hello world\n"),
			wantPubKey:  "pubkey",
			wantSig:     "sig",
			wantContent: []byte("hello world"),
			wantErr:     false,
		},
		{
			name:        "leading spaces before pubkey",
			input:       []byte("   pubkey sig message\n"),
			wantPubKey:  "pubkey",
			wantSig:     "sig",
			wantContent: []byte("message"),
			wantErr:     false,
		},
		{
			name:        "empty message content",
			input:       []byte("pubkey sig \n"),
			wantPubKey:  "pubkey",
			wantSig:     "sig",
			wantContent: []byte{},
			wantErr:     false,
		},
		{
			name:        "message with internal spaces preserved",
			input:       []byte("k s a  b   c\n"),
			wantPubKey:  "k",
			wantSig:     "s",
			wantContent: []byte("a  b   c"),
			wantErr:     false,
		},
		{
			name:        "base64-like pubkey and sig",
			input:       []byte("YWJjZGVm ZA==dGhpcw== message with spaces\n"),
			wantPubKey:  "YWJjZGVm",
			wantSig:     "ZA==dGhpcw==",
			wantContent: []byte("message with spaces"),
			wantErr:     false,
		},
		{
			name:        "message with unicode (UTF-8)",
			input:       []byte("pk sig Привет мир! 🚀\n"),
			wantPubKey:  "pk",
			wantSig:     "sig",
			wantContent: []byte("Привет мир! 🚀"),
			wantErr:     false,
		},
		{
			name:        "single char fields",
			input:       []byte("a b c\n"),
			wantPubKey:  "a",
			wantSig:     "b",
			wantContent: []byte("c"),
			wantErr:     false,
		},
		// длинное сообщение в пределах лимита
		{
			name: "long message within limit",
			input: func() []byte {
				content := bytes.Repeat([]byte("x"), 2000)
				return append([]byte("pk sig "), append(content, '\n')...)
			}(),
			wantPubKey:  "pk",
			wantSig:     "sig",
			wantContent: bytes.Repeat([]byte("x"), 2000),
			wantErr:     false,
		},

		// Кейс: сообщение больше лимита (должно отбиться мидлваром, но парсер тоже должен корректно обработать)
		{
			name: "message exceeds limit by 1 byte",
			input: func() []byte {
				content := bytes.Repeat([]byte("z"), 4089) // 7 + 4089 + 1 = 4097
				return append([]byte("pk sig "), append(content, '\n')...)
			}(),
			// Парсер технически сможет распарсить, но в реальном запросе это отловит MaxBytesHandler
			wantPubKey:  "pk",
			wantSig:     "sig",
			wantContent: bytes.Repeat([]byte("z"), 4089),
			wantErr:     false,
		},
		{
			name:        "message with tabs and special chars",
			input:       []byte("pk sig hello\tworld\n"),
			wantPubKey:  "pk",
			wantSig:     "sig",
			wantContent: []byte("hello\tworld"),
			wantErr:     false,
		},

		// Ошибки валидации
		{
			name:        "empty input",
			input:       []byte{},
			wantErr:     true,
			errContains: "missing protocol terminator",
		},
		{
			name:        "only newline",
			input:       []byte("\n"),
			wantErr:     true,
			errContains: "missing pubkey delimiter",
		},
		{
			name:        "only CRLF",
			input:       []byte("\r\n"),
			wantErr:     true,
			errContains: "missing pubkey delimiter",
		},
		{
			name:        "missing final newline",
			input:       []byte("pk sig message"),
			wantErr:     true,
			errContains: "missing protocol terminator",
		},
		{
			name:        "missing space after pubkey",
			input:       []byte("pubkeysig message\n"),
			wantErr:     true,
			errContains: "missing signature delimiter",
		},
		{
			name:        "missing signature",
			input:       []byte("pubkey \n"),
			wantErr:     true,
			errContains: "missing signature",
		},
		{
			name:        "missing space after signature",
			input:       []byte("pk sigmessage\n"),
			wantErr:     true,
			errContains: "missing signature delimiter",
		},
		{
			name:        "trailing spaces after sig before message",
			input:       []byte("pk sig    message\n"),
			wantPubKey:  "pk",
			wantSig:     "sig",
			wantContent: []byte("message"),
			wantErr:     false,
		},
		{
			name:        "message containing equals and slashes (base64 chars)",
			input:       []byte("pk sig a/b+c=d\n"),
			wantPubKey:  "pk",
			wantSig:     "sig",
			wantContent: []byte("a/b+c=d"),
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotPubKey, gotSig, gotContent, gotErr := handler.ParsePostBody(tt.input)

			if (gotErr != nil) != tt.wantErr {
				t.Errorf("parsePostBody() error = %v, wantErr %v", gotErr, tt.wantErr)
			}
			if tt.wantErr && tt.errContains != "" && gotErr != nil {
				if !errors.Is(gotErr, errors.New(tt.errContains)) &&
					!bytes.Contains([]byte(gotErr.Error()), []byte(tt.errContains)) {
					t.Errorf("error = %q, want substring %q", gotErr.Error(), tt.errContains)
				}
			}
			if gotPubKey != tt.wantPubKey {
				t.Errorf("pubKey = %q, want %q", gotPubKey, tt.wantPubKey)
			}
			if gotSig != tt.wantSig {
				t.Errorf("sig = %q, want %q", gotSig, tt.wantSig)
			}
			if !bytes.Equal(gotContent, tt.wantContent) {
				t.Errorf("content = %q, want %q", gotContent, tt.wantContent)
			}
		})
	}
}

// Fuzz-тест для поиска краевых случаев (Go 1.18+)
func FuzzParsePostBody(f *testing.F) {
	// Seed corpus с типичными кейсами
	f.Add([]byte("pk sig hello\n"))
	f.Add([]byte("a b c\r\n"))
	f.Add([]byte("key  sig   msg with spaces\n"))
	f.Add([]byte("pk sig \n")) // empty content
	f.Add([]byte("pk sig\n"))  // minimal valid

	f.Fuzz(func(t *testing.T, input []byte) {
		// Проверяем, что функция не паникует на любых входных данных
		_, _, _, err := handler.ParsePostBody(input)

		// Если вход валидный (заканчивается на \n), результат должен быть детерминированным
		// Если невалидный — должна возвращаться ошибка, а не паника
		_ = err // подавляем unused warning
	})
}

// Бенчмарк для контроля производительности парсера
func BenchmarkParsePostBody(b *testing.B) {
	// Типичный пакет: base64-ключи + сообщение ~100 байт
	input := []byte("YWJjZGVmZ2hpamtsbW5vcA== c2lnbmF0dXJlX2hlcmU= Hello, this is a test message for benchmarking!\n")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _, _, err := handler.ParsePostBody(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Тест на идемпотентность: парсинг → сборка → парсинг даёт тот же content
func TestParsePostBody_Idempotent(t *testing.T) {
	t.Parallel()

	original := []byte("testkey testsig This is a test message with spaces!\n")

	pk, sig, content, err := handler.ParsePostBody(original)
	if err != nil {
		t.Fatal(err)
	}

	// Собираем обратно (упрощённо, без учёта множественных пробелов)
	rebuilt := append([]byte(pk+" "+sig+" "), content...)
	rebuilt = append(rebuilt, '\n')

	// Парсим снова
	pk2, sig2, content2, err2 := handler.ParsePostBody(rebuilt)
	if err2 != nil {
		t.Fatal(err2)
	}

	if pk != pk2 || sig != sig2 || !bytes.Equal(content, content2) {
		t.Errorf("idempotency failed:\n  original: (%q,%q,%q)\n  rebuilt:  (%q,%q,%q)",
			pk, sig, content, pk2, sig2, content2)
	}
}
