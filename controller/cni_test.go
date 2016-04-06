package controller_test

import (
	"github.com/appc/cni/libcni"
	"github.com/cloudfoundry-incubator/guardian-cni-adapter/controller"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CNI", func() {

	Describe("AppendNetworkSpec", func() {
		var (
			networkSpec    string
			existingConfig *libcni.NetworkConfig
		)

		BeforeEach(func() {
			networkSpec = `{"key": "value"}`
			existingConfig = &libcni.NetworkConfig{
				Network: nil,
				Bytes:   []byte(`{"something": "some-value"}`),
			}
		})

		It("appends the garden network spec", func() {
			newNetworkSpec, err := controller.AppendNetworkSpec(existingConfig, networkSpec)
			Expect(err).NotTo(HaveOccurred())

			Expect(newNetworkSpec.Bytes).To(MatchJSON([]byte(`{"something":"some-value","network":{"key":"value"}}`)))
		})

		Context("when the network spec is empty", func() {
			It("should return an empty network field", func() {
				newNetworkSpec, err := controller.AppendNetworkSpec(existingConfig, "")
				Expect(err).NotTo(HaveOccurred())

				Expect(newNetworkSpec.Bytes).To(MatchJSON([]byte(`{"something":"some-value","network":""}`)))
			})
		})

		Context("when the existingNetConfig.Bytes is malformed JSON", func() {
			It("should return an error", func() {
				existingConfig.Bytes = []byte("%%%%%%")
				_, err := controller.AppendNetworkSpec(existingConfig, networkSpec)
				Expect(err).To(MatchError(ContainSubstring("unmarshal existing network bytes")))
			})
		})

		Context("when the network spec is malformed JSON", func() {
			It("should return an error", func() {
				_, err := controller.AppendNetworkSpec(existingConfig, "%%%%%%")
				Expect(err).To(MatchError(ContainSubstring("unmarshal garden network spec")))
			})
		})
	})
})
