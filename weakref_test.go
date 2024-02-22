package ason

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewWeakRef(t *testing.T) {
	t.Run("valid: creation with value", func(t *testing.T) {
		node := &Node{
			Ref:  1,
			Type: NodeTypeIdent,
		}
		weakRef := NewWeakRef(node)
		assert.NotNil(t, weakRef, "unexpected: weakRef is nil")
	})

	t.Run("invalid: creation with NIL value", func(t *testing.T) {
		weakRef := NewWeakRef(nil)
		assert.Nil(t, weakRef, "unexpected: ref should be nil")
	})
}

func TestWeakRef(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		node := &Node{
			Ref:  1,
			Type: NodeTypeIdent,
		}

		weakRef := NewWeakRef(node)
		assert.NotNil(t, weakRef.Load(), "unexpected: weakRef.Get() is nil")
		assert.True(t, weakRef.IsAlive(), "unexpected: weakRef.IsAlive() is false")
		assert.NotPanics(t, func() { t.Log(weakRef.Load().(*Node)) }, "unexpected: type conversion failed")
	})

	t.Run("invalid: forced cleanup by garbage collector", func(t *testing.T) {
		node := &Node{
			Ref:  1,
			Type: NodeTypeIdent,
		}
		weakRef := NewWeakRef(node)

		for i := 1; i < 10; i++ {
			runtime.Gosched()
			runtime.GC()
		}

		assert.NotNil(t, weakRef, "weakRef is nil after GC")
		assert.Nil(t, weakRef.Load(), "unexpected: weakRef.Get() is not nil after GC")
		assert.False(t, weakRef.IsAlive(), "unexpected: weakRef.IsAlive() is true")
	})

	t.Run("invalid: type conversion to a wrong type", func(t *testing.T) {
		node := &Node{
			Ref:  1,
			Type: NodeTypeIdent,
		}

		weakRef := NewWeakRef(node)
		assert.NotNil(t, weakRef.Load(), "unexpected: weakRef.Get() is nil")
		assert.True(t, weakRef.IsAlive(), "unexpected: weakRef.IsAlive() is false")
		assert.Panics(t, func() { t.Log(weakRef.Load().(*Ident)) }, "unexpected: type conversion should not work with wrong type")
	})

	t.Run("invalid: creation with NIL value", func(t *testing.T) {
		weakRef := NewWeakRef(nil)
		assert.Panics(t, func() { weakRef.Load() }, "unexpected: GetTarget() should be with panic if value was nil")
		assert.Panics(t, func() { weakRef.IsAlive() }, "unexpected: IsAlive() should be with panic if value was nil")
		assert.Panics(t, func() { t.Log(weakRef.Load().(*Node)) }, "unexpected: type conversion should not work")
	})
}
