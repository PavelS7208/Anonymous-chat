package service

import "errors"

type MessageData struct {
	PubKeyB64 string
	SigB64    string
	Message   string
}

// Validate проверяет базовую целостность полей
func (m *MessageData) Validate() error {
	if m.PubKeyB64 == "" {
		return errors.New("empty public key")
	}
	if m.SigB64 == "" {
		return errors.New("empty signature")
	}
	if m.Message == "" {
		return errors.New("empty message")
	}
	return nil
}
