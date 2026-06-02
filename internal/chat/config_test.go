package chat

import "testing"

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name:    "valid default",
			cfg:     DefaultConfig(),
			wantErr: false,
		},
		{
			name: "history length > storage",
			cfg: Config{
				initialHistoryLength:    2000,
				maxHistoryStorageLength: 1000,
				initialHistoryCap:       2000,
				maxMemberID:             100,
				eventChannelBuf:         64,
			},
			wantErr: true,
		},
		{
			name: "zero eventChannelBuf",
			cfg: Config{
				initialHistoryLength:    10,
				maxHistoryStorageLength: 1000,
				initialHistoryCap:       2000,
				maxMemberID:             100,
				eventChannelBuf:         0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
