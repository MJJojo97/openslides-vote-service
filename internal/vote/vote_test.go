package vote_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/OpenSlides/openslides-vote-service/internal/vote"
)

func TestVoteStart(t *testing.T) {
	backend := new(testBackend)
	v := vote.New(backend, backend, backend)

	t.Run("Unknown poll", func(t *testing.T) {
		if err := v.Start(1, vote.PollConfig{ContentObject: "motion/1"}); err != nil {
			t.Errorf("Start returned unexpected error: %v", err)
		}

		var gotConfig vote.PollConfig
		if err := json.Unmarshal(backend.config[1], &gotConfig); err != nil {
			t.Fatalf("Found invalid config in backend `%s`: %v", backend.config[1], err)
		}

		if gotConfig.ContentObject != "motion/1" {
			t.Errorf("Start created poll content_object `%s`, expected `motion/1`", gotConfig.ContentObject)
		}
	})

	t.Run("Known poll with same config", func(t *testing.T) {
		if err := v.Start(1, vote.PollConfig{ContentObject: "motion/1"}); err != nil {
			t.Errorf("Start returned unexpected error: %v", err)
		}

		var gotConfig vote.PollConfig
		if err := json.Unmarshal(backend.config[1], &gotConfig); err != nil {
			t.Fatalf("Found invalid config in backend `%s`: %v", backend.config[1], err)
		}

		if gotConfig.ContentObject != "motion/1" {
			t.Errorf("Start created poll content_object `%s`, expected `motion/1`", gotConfig.ContentObject)
		}
	})

	t.Run("Known poll with different config", func(t *testing.T) {
		err := v.Start(1, vote.PollConfig{ContentObject: "assignment/1"})

		if err == nil {
			t.Fatalf("Start did not return an error, expected one.")
		}

		var errTyped vote.TypeError
		if !errors.As(err, &errTyped) {
			t.Fatalf("Start did not return an Typed error. Got : %v", err)
		}

		if errTyped != vote.ErrExists {
			t.Fatalf("Got error of type `%s`, expected `errExists`", errTyped.Type())
		}
	})
}

func TestVoteStop(t *testing.T) {
	backend := new(testBackend)
	v := vote.New(backend, backend, backend)

	t.Run("Unknown poll", func(t *testing.T) {
		buf := new(bytes.Buffer)
		err := v.Stop(1, buf)

		var errTyped vote.TypeError
		if !errors.As(err, &errTyped) {
			t.Fatalf("Start did not return an Typed error. Got : %v", err)
		}

		if errTyped != vote.ErrNotExists {
			t.Errorf("Got error type `%s`, expected `not-exist`", errTyped.Type())
		}

		if buf.Len() != 0 {
			t.Errorf("Stop returned `%s`, expected no data", buf.String())
		}
	})

	t.Run("Known poll", func(t *testing.T) {
		if err := backend.Start(context.Background(), 1); err != nil {
			t.Fatalf("Starting poll: %v", err)
		}

		backend.objects[1] = [][]byte{
			[]byte("polldata1"),
			[]byte("polldata2"),
		}

		buf := new(bytes.Buffer)
		if err := v.Stop(1, buf); err != nil {
			t.Fatalf("Stop returned unexpected error: %v", err)
		}

		expect := [][]byte{
			[]byte("polldata1"),
			[]byte("polldata2"),
		}

		if got := buf.Bytes(); !reflect.DeepEqual(got, expect) {
			t.Errorf("Stop wrote `%s`, expected `%s`", got, expect)
		}
	})
}

func TestVoteVote(t *testing.T) {
	backend := new(testBackend)
	v := vote.New(backend, backend, backend)

	t.Run("Unknown poll", func(t *testing.T) {
		err := v.Vote(1, strings.NewReader(`{}`))

		var errTyped vote.TypeError
		if !errors.As(err, &errTyped) {
			t.Fatalf("Start did not return an Typed error. Got : %v", err)
		}

		if errTyped != vote.ErrNotExists {
			t.Errorf("Got error type `%s`, expected `not-exist`", errTyped.Type())
		}
	})

	t.Run("Invalid json", func(t *testing.T) {
		if err := backend.Start(context.Background(), 1); err != nil {
			t.Fatalf("Starting poll: %v", err)
		}

		err := v.Vote(1, strings.NewReader(`{123`))

		var errTyped vote.TypeError
		if !errors.As(err, &errTyped) {
			t.Fatalf("Vote() did not return an TypeError, got: %v", err)
		}

		if errTyped != vote.ErrVoteInvalid {
			t.Errorf("Got error type `%s`, expected `%s`", errTyped.Type(), vote.ErrVoteInvalid.Type())
		}
	})

	t.Run("Invalid format", func(t *testing.T) {
		if err := backend.Start(context.Background(), 1); err != nil {
			t.Fatalf("Starting poll: %v", err)
		}

		err := v.Vote(1, strings.NewReader(`{}`))

		var errTyped vote.TypeError
		if !errors.As(err, &errTyped) {
			t.Fatalf("Vote() did not return an TypeError, got: %v", err)
		}

		if errTyped != vote.ErrVoteInvalid {
			t.Errorf("Got error type `%s`, expected `%s`", errTyped.Type(), vote.ErrVoteInvalid.Type())
		}
	})

	t.Run("Valid motion data", func(t *testing.T) {

	})

	t.Run("Valid assignment data", func(t *testing.T) {

	})
}

// testBackend is a simple (not concurent) vote backend that can be used for
// testing.
type testBackend struct {
	config  map[int][]byte
	voted   map[int]map[int]bool
	objects map[int][][]byte
	stopped map[int]bool
}

func (b *testBackend) Start(ctx context.Context, pollID int) error {
	if b.config == nil {
		b.config = make(map[int][]byte)
		b.voted = make(map[int]map[int]bool)
		b.objects = make(map[int][][]byte)
		b.stopped = make(map[int]bool)
	}

	b.voted[pollID] = make(map[int]bool)
	return nil
}

func (b *testBackend) SetConfig(ctx context.Context, pollID int, config []byte) error {
	b.config[pollID] = config
	return nil
}

func (b *testBackend) Config(ctx context.Context, pollID int) ([]byte, error) {
	config, ok := b.config[pollID]
	if !ok {
		return nil, fmt.Errorf("unknown poll with id %d", pollID)
	}
	return config, nil
}

func (b *testBackend) Vote(ctx context.Context, pollID int, userID int, object []byte) error {
	if _, ok := b.config[pollID]; !ok {
		return fmt.Errorf("unknown poll with id %d", pollID)
	}

	if b.stopped[pollID] {
		return fmt.Errorf("Poll is stopped")
	}

	if _, ok := b.voted[pollID][userID]; ok {
		return fmt.Errorf("user has already voted")
	}

	b.voted[pollID][userID] = true
	b.objects[pollID] = append(b.objects[pollID], object)
	return nil
}

func (b *testBackend) Stop(ctx context.Context, pollID int) ([][]byte, error) {
	b.stopped[pollID] = true
	return b.objects[pollID], nil
}

func (b *testBackend) Clear(ctx context.Context, pollID int) error {
	delete(b.config, pollID)
	delete(b.voted, pollID)
	delete(b.objects, pollID)
	delete(b.stopped, pollID)
	return nil
}
