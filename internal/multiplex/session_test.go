package multiplex

import (
	"bytes"
	"github.com/cbeuw/Cloak/internal/util"
	"math/rand"
	"strconv"
	"sync/atomic"
	"testing"
)

var seshConfigOrdered = &SessionConfig{
	Obfuscator: nil,
	Valve:      nil,
	UnitRead:   util.ReadTLS,
}

var seshConfigUnordered = &SessionConfig{
	Obfuscator: nil,
	Valve:      nil,
	UnitRead:   util.ReadTLS,
	Unordered:  true,
}

func TestRecvDataFromRemote(t *testing.T) {
	testPayloadLen := 1024
	testPayload := make([]byte, testPayloadLen)
	rand.Read(testPayload)
	f := &Frame{
		1,
		0,
		0,
		testPayload,
	}
	obfsBuf := make([]byte, 17000)

	sessionKey := make([]byte, 32)
	rand.Read(sessionKey)
	t.Run("plain ordered", func(t *testing.T) {
		obfuscator, _ := GenerateObfs(E_METHOD_PLAIN, sessionKey, true)
		seshConfigOrdered.Obfuscator = obfuscator
		sesh := MakeSession(0, seshConfigOrdered)
		n, _ := sesh.Obfs(f, obfsBuf)

		err := sesh.recvDataFromRemote(obfsBuf[:n])
		if err != nil {
			t.Error(err)
			return
		}
		stream, err := sesh.Accept()
		if err != nil {
			t.Error(err)
			return
		}

		resultPayload := make([]byte, testPayloadLen)
		_, err = stream.Read(resultPayload)
		if err != nil {
			t.Error(err)
			return
		}
		if !bytes.Equal(testPayload, resultPayload) {
			t.Errorf("Expecting %x, got %x", testPayload, resultPayload)
		}
	})
	t.Run("aes-gcm ordered", func(t *testing.T) {
		obfuscator, _ := GenerateObfs(E_METHOD_AES_GCM, sessionKey, true)
		seshConfigOrdered.Obfuscator = obfuscator
		sesh := MakeSession(0, seshConfigOrdered)
		n, _ := sesh.Obfs(f, obfsBuf)

		err := sesh.recvDataFromRemote(obfsBuf[:n])
		if err != nil {
			t.Error(err)
			return
		}
		stream, err := sesh.Accept()
		if err != nil {
			t.Error(err)
			return
		}

		resultPayload := make([]byte, testPayloadLen)
		_, err = stream.Read(resultPayload)
		if err != nil {
			t.Error(err)
			return
		}
		if !bytes.Equal(testPayload, resultPayload) {
			t.Errorf("Expecting %x, got %x", testPayload, resultPayload)
		}
	})
	t.Run("chacha20-poly1305 ordered", func(t *testing.T) {
		obfuscator, _ := GenerateObfs(E_METHOD_CHACHA20_POLY1305, sessionKey, true)
		seshConfigOrdered.Obfuscator = obfuscator
		sesh := MakeSession(0, seshConfigOrdered)
		n, _ := sesh.Obfs(f, obfsBuf)

		err := sesh.recvDataFromRemote(obfsBuf[:n])
		if err != nil {
			t.Error(err)
			return
		}
		stream, err := sesh.Accept()
		if err != nil {
			t.Error(err)
			return
		}

		resultPayload := make([]byte, testPayloadLen)
		_, err = stream.Read(resultPayload)
		if err != nil {
			t.Error(err)
			return
		}
		if !bytes.Equal(testPayload, resultPayload) {
			t.Errorf("Expecting %x, got %x", testPayload, resultPayload)
		}
	})

	t.Run("plain unordered", func(t *testing.T) {
		obfuscator, _ := GenerateObfs(E_METHOD_PLAIN, sessionKey, true)
		seshConfigUnordered.Obfuscator = obfuscator
		sesh := MakeSession(0, seshConfigOrdered)
		n, _ := sesh.Obfs(f, obfsBuf)

		err := sesh.recvDataFromRemote(obfsBuf[:n])
		if err != nil {
			t.Error(err)
			return
		}
		stream, err := sesh.Accept()
		if err != nil {
			t.Error(err)
			return
		}

		resultPayload := make([]byte, testPayloadLen)
		_, err = stream.Read(resultPayload)
		if err != nil {
			t.Error(err)
			return
		}
		if !bytes.Equal(testPayload, resultPayload) {
			t.Errorf("Expecting %x, got %x", testPayload, resultPayload)
		}
	})
}

func TestRecvDataFromRemote_Closing_InOrder(t *testing.T) {
	testPayloadLen := 1024
	testPayload := make([]byte, testPayloadLen)
	rand.Read(testPayload)
	obfsBuf := make([]byte, 17000)

	sessionKey := make([]byte, 32)
	obfuscator, _ := GenerateObfs(E_METHOD_PLAIN, sessionKey, true)
	seshConfigOrdered.Obfuscator = obfuscator

	rand.Read(sessionKey)
	sesh := MakeSession(0, seshConfigOrdered)

	f1 := &Frame{
		1,
		0,
		C_NOOP,
		testPayload,
	}
	// create stream 1
	n, _ := sesh.Obfs(f1, obfsBuf)
	err := sesh.recvDataFromRemote(obfsBuf[:n])
	if err != nil {
		t.Fatalf("receiving normal frame for stream 1: %v", err)
	}
	s1I, ok := sesh.streams.Load(f1.StreamID)
	if !ok {
		t.Fatal("failed to fetch stream 1 after receiving it")
	}
	if sesh.streamCount() != 1 {
		t.Error("stream count isn't 1")
	}

	// create stream 2
	f2 := &Frame{
		2,
		0,
		C_NOOP,
		testPayload,
	}
	n, _ = sesh.Obfs(f2, obfsBuf)
	err = sesh.recvDataFromRemote(obfsBuf[:n])
	if err != nil {
		t.Fatalf("receiving normal frame for stream 2: %v", err)
	}
	s2I, ok := sesh.streams.Load(f2.StreamID)
	if s2I == nil || !ok {
		t.Fatal("failed to fetch stream 2 after receiving it")
	}
	if sesh.streamCount() != 2 {
		t.Error("stream count isn't 2")
	}

	// close stream 1
	f1CloseStream := &Frame{
		1,
		1,
		C_STREAM,
		testPayload,
	}
	n, _ = sesh.Obfs(f1CloseStream, obfsBuf)
	err = sesh.recvDataFromRemote(obfsBuf[:n])
	if err != nil {
		t.Fatalf("receiving stream closing frame for stream 1: %v", err)
	}
	s1I, _ = sesh.streams.Load(f1.StreamID)
	if s1I != nil {
		t.Fatal("stream 1 still exist after receiving stream close")
	}
	s1, _ := sesh.Accept()
	if !s1.(*Stream).isClosed() {
		t.Fatal("stream 1 not marked as closed")
	}
	payloadBuf := make([]byte, testPayloadLen)
	_, err = s1.Read(payloadBuf)
	if err != nil || !bytes.Equal(payloadBuf, testPayload) {
		t.Fatalf("failed to read from stream 1 after closing: %v", err)
	}
	s2, _ := sesh.Accept()
	if s2.(*Stream).isClosed() {
		t.Fatal("stream 2 shouldn't be closed")
	}
	if sesh.streamCount() != 1 {
		t.Error("stream count isn't 1 after stream 1 closed")
	}

	// close stream 1 again
	n, _ = sesh.Obfs(f1CloseStream, obfsBuf)
	err = sesh.recvDataFromRemote(obfsBuf[:n])
	if err != nil {
		t.Fatalf("receiving stream closing frame for stream 1 %v", err)
	}
	s1I, _ = sesh.streams.Load(f1.StreamID)
	if s1I != nil {
		t.Error("stream 1 exists after receiving stream close for the second time")
	}
	if sesh.streamCount() != 1 {
		t.Error("stream count isn't 1 after stream 1 closed twice")
	}

	// close session
	fCloseSession := &Frame{
		StreamID: 0xffffffff,
		Seq:      0,
		Closing:  C_SESSION,
		Payload:  testPayload,
	}
	n, _ = sesh.Obfs(fCloseSession, obfsBuf)
	err = sesh.recvDataFromRemote(obfsBuf[:n])
	if err != nil {
		t.Fatalf("receiving session closing frame: %v", err)
	}
	if !sesh.IsClosed() {
		t.Error("session not closed after receiving signal")
	}
	if !s2.(*Stream).isClosed() {
		t.Error("stream 2 isn't closed after session closed")
	}
	if _, err := s2.Read(payloadBuf); err != nil || !bytes.Equal(payloadBuf, testPayload) {
		t.Error("failed to read from stream 2 after session closed")
	}
	if _, err := s2.Write(testPayload); err == nil {
		t.Error("can still write to stream 2 after session closed")
	}
	if sesh.streamCount() != 0 {
		t.Error("stream count isn't 0 after session closed")
	}
}

func TestRecvDataFromRemote_Closing_OutOfOrder(t *testing.T) {
	// Tests for when the closing frame of a stream is received first before any data frame
	testPayloadLen := 1024
	testPayload := make([]byte, testPayloadLen)
	rand.Read(testPayload)
	obfsBuf := make([]byte, 17000)

	sessionKey := make([]byte, 32)
	obfuscator, _ := GenerateObfs(E_METHOD_PLAIN, sessionKey, true)
	seshConfigOrdered.Obfuscator = obfuscator

	rand.Read(sessionKey)
	sesh := MakeSession(0, seshConfigOrdered)

	// receive stream 1 closing first
	f1CloseStream := &Frame{
		1,
		1,
		C_STREAM,
		testPayload,
	}
	n, _ := sesh.Obfs(f1CloseStream, obfsBuf)
	err := sesh.recvDataFromRemote(obfsBuf[:n])
	if err != nil {
		t.Fatalf("receiving out of order stream closing frame for stream 1: %v", err)
	}
	_, ok := sesh.streams.Load(f1CloseStream.StreamID)
	if !ok {
		t.Fatal("stream 1 doesn't exist")
	}
	if sesh.streamCount() != 1 {
		t.Error("stream count isn't 1 after stream 1 received")
	}

	// receive data frame of stream 1 after receiving the closing frame
	f1 := &Frame{
		1,
		0,
		C_NOOP,
		testPayload,
	}
	n, _ = sesh.Obfs(f1, obfsBuf)
	err = sesh.recvDataFromRemote(obfsBuf[:n])
	if err != nil {
		t.Fatalf("receiving normal frame for stream 1: %v", err)
	}
	s1, err := sesh.Accept()
	if err != nil {
		t.Fatal("failed to accept stream 1 after receiving it")
	}
	payloadBuf := make([]byte, testPayloadLen)
	if _, err := s1.Read(payloadBuf); err != nil || !bytes.Equal(payloadBuf, testPayload) {
		t.Error("failed to read from steam 1")
	}
	if !s1.(*Stream).isClosed() {
		t.Error("s1 isn't closed")
	}
	if sesh.streamCount() != 0 {
		t.Error("stream count isn't 0 after stream 1 closed")
	}
}

func TestParallel(t *testing.T) {
	rand.Seed(0)

	sessionKey := make([]byte, 32)
	obfuscator, _ := GenerateObfs(E_METHOD_PLAIN, sessionKey, true)
	seshConfigOrdered.Obfuscator = obfuscator
	rand.Read(sessionKey)
	sesh := MakeSession(0, seshConfigOrdered)

	numStreams := 10
	seqs := make([]*uint64, numStreams)
	for i, _ := range seqs {
		seqs[i] = new(uint64)
	}
	randFrame := func() *Frame {
		id := rand.Intn(numStreams)
		return &Frame{
			uint32(id),
			atomic.AddUint64(seqs[id], 1) - 1,
			uint8(rand.Intn(2)),
			[]byte{1, 2, 3, 4},
		}
	}

	numOfTests := 100
	tests := make([]struct {
		name  string
		frame *Frame
	}, numOfTests)
	for i, _ := range tests {
		tests[i].name = strconv.Itoa(i)
		tests[i].frame = randFrame()
	}

	for _, tc := range tests {
		go func(frame *Frame) {
			data := make([]byte, 1000)
			n, _ := sesh.Obfs(frame, data)
			data = data[0:n]

			err := sesh.recvDataFromRemote(data)
			if err != nil {
				t.Error(err)
			}
		}(tc.frame)
	}

	var count int
	sesh.streams.Range(func(_, s interface{}) bool {
		if s != nil {
			count++
		}
		return true
	})
	sc := int(sesh.streamCount())
	if sc != count {
		t.Errorf("broken referential integrety: actual %v, reference count: %v", count, sc)
	}
}

func BenchmarkRecvDataFromRemote_Ordered(b *testing.B) {
	testPayloadLen := 1024
	testPayload := make([]byte, testPayloadLen)
	rand.Read(testPayload)
	f := &Frame{
		1,
		0,
		0,
		testPayload,
	}
	obfsBuf := make([]byte, 17000)

	sessionKey := make([]byte, 32)
	rand.Read(sessionKey)

	b.Run("plain", func(b *testing.B) {
		obfuscator, _ := GenerateObfs(E_METHOD_PLAIN, sessionKey, true)
		seshConfigOrdered.Obfuscator = obfuscator
		sesh := MakeSession(0, seshConfigOrdered)
		n, _ := sesh.Obfs(f, obfsBuf)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := sesh.recvDataFromRemote(obfsBuf[:n])
			if err != nil {
				b.Error(err)
				return
			}
			b.SetBytes(int64(n))
		}
	})

	b.Run("aes-gcm", func(b *testing.B) {
		obfuscator, _ := GenerateObfs(E_METHOD_AES_GCM, sessionKey, true)
		seshConfigOrdered.Obfuscator = obfuscator
		sesh := MakeSession(0, seshConfigOrdered)
		n, _ := sesh.Obfs(f, obfsBuf)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := sesh.recvDataFromRemote(obfsBuf[:n])
			if err != nil {
				b.Error(err)
				return
			}
			b.SetBytes(int64(n))
		}
	})

	b.Run("chacha20-poly1305", func(b *testing.B) {
		obfuscator, _ := GenerateObfs(E_METHOD_CHACHA20_POLY1305, sessionKey, true)
		seshConfigOrdered.Obfuscator = obfuscator
		sesh := MakeSession(0, seshConfigOrdered)
		n, _ := sesh.Obfs(f, obfsBuf)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := sesh.recvDataFromRemote(obfsBuf[:n])
			if err != nil {
				b.Error(err)
				return
			}
			b.SetBytes(int64(n))
		}
	})
}
