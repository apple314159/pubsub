// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package pubsub_test

import (
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/pubsub"
)

type MultiplexerHubSuite struct{}

var _ = gc.Suite(&MultiplexerHubSuite{})

func (*MultiplexerHubSuite) TestNewMultiplexerStructuredHub(c *gc.C) {
	hub := pubsub.NewStructuredHub(nil)
	multi, err := hub.NewMultiplexer()
	c.Assert(err, jc.ErrorIsNil)
	defer multi.Unsubscribe()
	c.Check(multi, gc.NotNil)
}

func (*MultiplexerHubSuite) TestMultiplexerAdd(c *gc.C) {
	hub := pubsub.NewStructuredHub(nil)
	multi, err := hub.NewMultiplexer()
	c.Assert(err, jc.ErrorIsNil)
	defer multi.Unsubscribe()
	for i, test := range []struct {
		description string
		handler     interface{}
		err         string
	}{
		{
			description: "nil handler",
			err:         "nil handler not valid",
		}, {
			description: "string handler",
			handler:     "a string",
			err:         "handler of type string not valid",
		}, {
			description: "too few args",
			handler:     func(string) {},
			err:         "expected 2 or 3 args, got 1, incorrect handler signature not valid",
		}, {
			description: "too many args",
			handler:     func(string, string, string, string) {},
			err:         "expected 2 or 3 args, got 4, incorrect handler signature not valid",
		}, {
			description: "simple hub handler function",
			handler:     func(string, interface{}) {},
			err:         "second arg should be a structure or map\\[string\\]interface{} for data, incorrect handler signature not valid",
		}, {
			description: "bad return values in handler function",
			handler:     func(string, interface{}, error) error { return nil },
			err:         "expected no return values, got 1, incorrect handler signature not valid",
		}, {
			description: "bad first arg",
			handler:     func(int, map[string]interface{}, error) {},
			err:         "first arg should be a string, incorrect handler signature not valid",
		}, {
			description: "bad second arg",
			handler:     func(string, string, error) {},
			err:         "second arg should be a structure or map\\[string\\]interface{} for data, incorrect handler signature not valid",
		}, {
			description: "bad third arg",
			handler:     func(string, map[string]interface{}, error) {},
			err:         "data type of map\\[string\\]interface{} expects only 2 args, got 3, incorrect handler signature not valid",
		}, {
			description: "accept map[string]interface{}",
			handler:     func(string, map[string]interface{}) {},
		}, {
			description: "bad map[string]string",
			handler:     func(string, map[string]string, error) {},
			err:         "second arg should be a structure or map\\[string\\]interface{} for data, incorrect handler signature not valid",
		}, {
			description: "bad third arg",
			handler:     func(string, Emitter, bool) {},
			err:         "third arg should be error for deserialization errors, incorrect handler signature not valid",
		}, {
			description: "accept struct value",
			handler:     func(string, Emitter, error) {},
		},
	} {
		c.Logf("test %d: %s", i, test.description)
		err := multi.AddMatch(pubsub.MatchAll, test.handler)
		if test.err == "" {
			c.Check(err, jc.ErrorIsNil)
		} else {
			c.Check(err, gc.ErrorMatches, test.err)
		}
	}
}

func (*MultiplexerHubSuite) TestMatcher(c *gc.C) {
	hub := pubsub.NewStructuredHub(nil)
	multi, err := hub.NewMultiplexer()
	c.Assert(err, jc.ErrorIsNil)
	defer multi.Unsubscribe()

	noopFunc := func(string, map[string]interface{}) {}
	err = multi.Add(first, noopFunc)
	c.Assert(err, jc.ErrorIsNil)
	err = multi.AddMatch(pubsub.MatchRegexp("second.*"), noopFunc)
	c.Assert(err, jc.ErrorIsNil)

	c.Check(pubsub.MultiplexerMatch(multi, first), jc.IsTrue)
	c.Check(pubsub.MultiplexerMatch(multi, firstdot), jc.IsFalse)
	c.Check(pubsub.MultiplexerMatch(multi, second), jc.IsTrue)
	c.Check(pubsub.MultiplexerMatch(multi, space), jc.IsFalse)
}

func (*MultiplexerHubSuite) TestCallback(c *gc.C) {
	source := Emitter{
		Origin:  "test",
		Message: "hello world",
		ID:      42,
	}
	var (
		topic         = "callback.topic"
		originCalled  bool
		messageCalled bool
		mapCalled     bool
	)
	hub := pubsub.NewStructuredHub(nil)
	multi, err := hub.NewMultiplexer()
	c.Assert(err, jc.ErrorIsNil)
	defer multi.Unsubscribe()

	err = multi.Add(topic, func(top string, data JustOrigin, err error) {
		c.Check(err, jc.ErrorIsNil)
		c.Check(top, gc.Equals, topic)
		c.Check(data.Origin, gc.Equals, source.Origin)
		originCalled = true
	})
	c.Assert(err, jc.ErrorIsNil)
	err = multi.Add(second, func(topic string, data MessageID, err error) {
		c.Fail()
		messageCalled = true
	})
	c.Assert(err, jc.ErrorIsNil)
	err = multi.AddMatch(pubsub.MatchAll, func(top string, data map[string]interface{}) {
		c.Check(top, gc.Equals, topic)
		c.Check(data, jc.DeepEquals, map[string]interface{}{
			"origin":  "test",
			"message": "hello world",
			"id":      float64(42), // ints are converted to floats through json.
		})
		mapCalled = true
	})
	c.Assert(err, jc.ErrorIsNil)
	done, err := hub.Publish(topic, source)
	c.Assert(err, jc.ErrorIsNil)

	waitForMessageHandlingToBeComplete(c, done)
	c.Check(originCalled, jc.IsTrue)
	c.Check(messageCalled, jc.IsFalse)
	c.Check(mapCalled, jc.IsTrue)
}
