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

	It("counts the frames showing a state per run", func() {
		b := behaviour.Analyze([]frames.Frame{
			{st("a"), st("a")},
			{st("a"), st("b")},
			{st("c"), st("b")},
		})
		Expect(b.StateFrames).To(Equal(map[string][]int{
			`"a" 0:-`: {2}, // a frame showing a state twice counts once
			`"b" 0:-`: {2},
			`"c" 0:-`: {1},
		}))
		b.Merge(behaviour.Analyze([]frames.Frame{{st("a")}}))
		b.Merge(behaviour.Analyze([]frames.Frame{{st("d")}, {st("d")}}))
		Expect(b.StateFrames).To(Equal(map[string][]int{
			`"a" 0:-`: {2, 1, 0},
			`"b" 0:-`: {2, 0, 0},
			`"c" 0:-`: {1, 0, 0},
			`"d" 0:-`: {0, 0, 2},
		}))
	})

	It("keeps unknown numbers of frames unknown when merging", func() {
		b := behaviour.New()
		b.Frames = []int{1}
		b.States[st("a").Key()] = st("a")
		b.Merge(behaviour.Analyze([]frames.Frame{{st("a")}, {st("b")}}))
		Expect(b.StateFrames).To(Equal(map[string][]int{`"b" 0:-`: {0, 1}}))
	})

	It("detects transient states", func() {
		b := behaviour.Analyze([]frames.Frame{{st("a")}, {st("b")}, {st("b")}, {st("c")}})
		b.Merge(behaviour.Analyze([]frames.Frame{{st("b")}, {st("c")}}))
		// shown in a single frame, but not in all runs
		Expect(b.Transient(st("a").Key())).To(BeTrue())
		// shown in several frames of a run
		Expect(b.Transient(st("b").Key())).To(BeFalse())
		// shown in all runs
		Expect(b.Transient(st("c").Key())).To(BeFalse())
		// unknown state
		Expect(b.Transient(st("x").Key())).To(BeFalse())
	})

	It("is not transient for a single run", func() {
		b := behaviour.Analyze([]frames.Frame{{st("a")}, {st("b")}})
		Expect(b.Transient(st("a").Key())).To(BeFalse())
	})

	It("removes states and their transitions", func() {
		b := behaviour.Analyze([]frames.Frame{{st("a")}, {st("b")}, {st("c")}})
		r := b.Without(st("b").Key())
		Expect(r.StateKeys()).To(Equal([]string{`"a" 0:-`, `"c" 0:-`}))
		Expect(r.TransitionKeys()).To(BeEmpty())
		Expect(r.StateFrames).To(HaveLen(2))
		Expect(r.Frames).To(Equal(b.Frames))
		// the original behaviour is not modified
		Expect(b.StateKeys()).To(HaveLen(3))
		Expect(b.TransitionKeys()).To(HaveLen(2))
	})
})
