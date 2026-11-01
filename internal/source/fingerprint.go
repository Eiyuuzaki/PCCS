package source

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

// Fingerprint 对字段做稳定指纹（按 key 排序后拼接），用于"变了才推"。
func Fingerprint(fields map[string]string) string {
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte(0)
		b.WriteString(fields[k])
		b.WriteByte(0)
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:8]) // 16 位十六进制
}