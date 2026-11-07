package main

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Command", func() {
	It("requires a command", func() {
		Expect(run(nil)).To(Equal(2))
	})

	It("rejects unknown commands", func() {
		Expect(run([]string{"unknown"})).To(Equal(2))
	})

	It("provides help", func() {
		Expect(run([]string{"help"})).To(Equal(0))
	})

	It("rejects invalid options", func() {
		Expect(run([]string{"check"})).To(Equal(2))
	})
})
