package main

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/mandelsoft/ttyframes/frames"
	"github.com/mandelsoft/ttyframes/runner"
)

var _ = Describe("Config", func() {
	It("provides defaults", func() {
		cfg, err := parseConfig("check", []string{"./..."})
		Expect(err).To(Succeed())
		Expect(cfg.runs).To(Equal(3))
		Expect(cfg.file).To(Equal(DefaultFile))
		Expect(cfg.masks).To(Equal(frames.DefaultMasks))
		Expect(cfg.options).To(Equal(runner.DefaultOptions))
		Expect(cfg.patterns).To(Equal([]string{"./..."}))
	})

	It("parses options", func() {
		cfg, err := parseConfig("record", []string{"-runs", "1", "-file", "x.ttyframes.json", "-parallel", "2", "-details", "5", "-v", "-strict",
			"-mask", `\d+ KB=N KB`, "-layout", "_2 January 2006", "-env", "A=b", "-cols", "80", "-rows", "24", "-timeout", "1m", "./a", "./b"})
		Expect(err).To(Succeed())
		Expect(cfg.runs).To(Equal(1))
		Expect(cfg.file).To(Equal("x.ttyframes.json"))
		Expect(cfg.parallel).To(Equal(2))
		Expect(cfg.details).To(Equal(5))
		Expect(cfg.verbose).To(BeTrue())
		Expect(cfg.strict).To(BeTrue())
		Expect(cfg.masks).To(HaveLen(len(frames.DefaultMasks) + 1))
		Expect(cfg.masks[len(cfg.masks)-1].String()).To(Equal(`\d+ KB=N KB`))
		Expect(cfg.options).To(Equal(runner.Options{Columns: 80, Rows: 24, Timeout: time.Minute, Env: []string{"A=b"}}))
		Expect(cfg.patterns).To(Equal([]string{"./a", "./b"}))
		Expect(cfg.layouts).To(Equal([]string{"_2 January 2006"}))
		Expect(cfg.comparer).NotTo(BeNil())
	})

	DescribeTable("rejects invalid arguments",
		func(args ...string) {
			_, err := parseConfig("check", args)
			Expect(err).To(HaveOccurred())
		},
		Entry("no patterns"),
		Entry("invalid runs", "-runs", "0", "."),
		Entry("invalid mask", "-mask", "x", "."),
		Entry("invalid environment", "-env", "x", "."),
		Entry("invalid layout", "-layout", "abc", "."),
		Entry("unknown option", "-unknown", "."),
	)
})
