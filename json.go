package qbittorrent

import "github.com/bytedance/sonic"

// jsonMarshal is a drop-in for sonic.Marshal, enabling easy swap-out.
func jsonMarshal(v any) ([]byte, error) {
	return sonic.Marshal(v)
}
