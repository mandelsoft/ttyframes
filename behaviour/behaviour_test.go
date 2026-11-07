package behaviour_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/mandelsoft/ttyframes/behaviour"
	"github.com/mandelsoft/ttyframes/frames"
)

// st provides a state with a single unformatted segment.
func st(text string) frames.State {
	return frames.State{{Text: text}}
}

var _ = Describe("Behaviour", func() {
	It("collects states and transitions", func() {
		b := behaviour.Analyze([]frames.Frame{
			{st("a 1"), st("b")},
			{st("a 2"), st("b")},
			{st("a 2"), st("b"), st("c")},
			{st("a 3"), st("b"), st("c")},
		})
		Expect(b.Frames).To(Equal([]int{4}))
		Expect(b.StateKeys()).To(Equal([]string{`"a 1" 0:-`, `"a 2" 0:-`, `"a 3" 0:-`, `"b" 0:-`, `"c" 0:-`}))
		// no transitions between frames with different numbers of lines
		Expect(b.TransitionKeys()).To(Equal([]string{
			`"a 1" 0:- -> "a 2" 0:-`,
			`"a 2" 0:- -> "a 3" 0:-`,
		}))
	})

	It("merges runs", func() {
		b := behaviour.Analyze([]frames.Frame{{st("a")}, {st("b")}})
		b.Merge(behaviour.Analyze([]frames.Frame{{st("a")}, {st("c")}}))
		Expect(b.Frames).To(Equal([]int{2, 2}))
		Expect(b.StateKeys()).To(HaveLen(3))
		Expect(b.TransitionKeys()).To(HaveLen(2))
	})
})
