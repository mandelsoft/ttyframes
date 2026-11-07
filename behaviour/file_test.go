package behaviour_test

import (
	"bytes"
	"encoding/json"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/mandelsoft/ttyframes/behaviour"
	"github.com/mandelsoft/ttyframes/frames"
)

var _ = Describe("File", func() {
	b := behaviour.Analyze([]frames.Frame{
		{frames.State{{Text: "Ns ", Format: "rev=7,fg=32"}, {Text: "äb\"<c"}}, nil},
		{frames.State{{Text: "Ns äb", Format: "rev=7,fg=32"}, {Text: "\"<c"}}, nil},
	})
	file := `{
  "format": "ttyframes/v1",
  "frames": [2],
  "states": [
    {"text":""},
    {"text":"Ns äb\"<c","runs":[{"start":0,"format":{"fg":"32","rev":"7"}},{"start":3}]},
    {"text":"Ns äb\"<c","runs":[{"start":0,"format":{"fg":"32","rev":"7"}},{"start":5}]}
  ],
  "transitions": [
    {"from":{"text":"Ns äb\"<c","runs":[{"start":0,"format":{"fg":"32","rev":"7"}},{"start":3}]},"to":{"text":"Ns äb\"<c","runs":[{"start":0,"format":{"fg":"32","rev":"7"}},{"start":5}]}}
  ]
}
`

	It("writes a behaviour with an element per line", func() {
		var buf bytes.Buffer
		Expect(behaviour.Write(&buf, b)).To(Succeed())
		Expect(buf.String()).To(Equal(file))
		Expect(json.Valid(buf.Bytes())).To(BeTrue())
	})

	It("writes an empty behaviour", func() {
		var buf bytes.Buffer
		Expect(behaviour.Write(&buf, behaviour.New())).To(Succeed())
		Expect(buf.String()).To(Equal("{\n  \"format\": \"ttyframes/v1\",\n  \"frames\": [],\n  \"states\": [],\n  \"transitions\": []\n}\n"))
		r, err := behaviour.Read(&buf)
		Expect(err).To(Succeed())
		Expect(r.States).To(BeEmpty())
	})

	It("reads a behaviour", func() {
		r, err := behaviour.Read(strings.NewReader(file))
		Expect(err).To(Succeed())
		Expect(r).To(Equal(b))
	})

	It("reads any valid layout", func() {
		var v any
		Expect(json.Unmarshal([]byte(file), &v)).To(Succeed())
		data, err := json.MarshalIndent(v, "", "\t")
		Expect(err).To(Succeed())
		r, err := behaviour.Read(bytes.NewReader(data))
		Expect(err).To(Succeed())
		Expect(r).To(Equal(b))
	})

	DescribeTable("rejects invalid files",
		func(content string) {
			_, err := behaviour.Read(strings.NewReader(content))
			Expect(err).To(HaveOccurred())
		},
		Entry("empty", ""),
		Entry("no JSON", "state \"a\" 0:-\n"),
		Entry("wrong format", `{"format":"other"}`),
		Entry("unknown attribute", `{"format":"ttyframes/v1","other":1}`),
		Entry("missing runs", `{"format":"ttyframes/v1","states":[{"text":"a"}]}`),
		Entry("run not at start", `{"format":"ttyframes/v1","states":[{"text":"ab","runs":[{"start":1}]}]}`),
		Entry("decreasing runs", `{"format":"ttyframes/v1","states":[{"text":"ab","runs":[{"start":0},{"start":1},{"start":1}]}]}`),
		Entry("run beyond text", `{"format":"ttyframes/v1","states":[{"text":"ab","runs":[{"start":0},{"start":3}]}]}`),
		Entry("runs for empty text", `{"format":"ttyframes/v1","states":[{"text":"","runs":[{"start":0}]}]}`),
		Entry("unknown area", `{"format":"ttyframes/v1","states":[{"text":"a","runs":[{"start":0,"format":{"color":"1"}}]}]}`),
		Entry("invalid transition", `{"format":"ttyframes/v1","transitions":[{"from":{"text":"a"},"to":{"text":"b","runs":[{"start":0}]}}]}`),
	)

	It("converts states", func() {
		s := frames.State{{Text: "a"}, {Text: "b", Format: "hl=1,ul=4:3"}}
		d := behaviour.NewState(s)
		Expect(d).To(Equal(behaviour.State{Text: "ab", Runs: []behaviour.Run{{Start: 0}, {Start: 1, Format: map[string]string{"hl": "1", "ul": "4:3"}}}}))
		r, err := d.State()
		Expect(err).To(Succeed())
		Expect(r).To(Equal(s))
	})
})
