package webssh

import (
	"bytes"
	log "github.com/sirupsen/logrus"
	"testing"
)

func TestSafeBuffer_Bytes(t *testing.T) {
	t.Run("bytes.Buffer", func(t *testing.T) {
		temp := new(bytes.Buffer)
		temp.Write([]byte("my test"))
		log.Printf("temp value = %v", temp)
		temp.Reset()
		log.Printf("temp value = %v", temp)
	})

	t.Run("safe.Buffer", func(t *testing.T) {
		temp := new(safeBuffer)
		temp.Write([]byte("my test"))
		log.Printf("safe value = %v", temp)
		temp.Reset()
		log.Printf("safe temp value = %v", temp)
	})
}
