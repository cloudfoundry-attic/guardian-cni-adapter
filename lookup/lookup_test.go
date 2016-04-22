package lookup_test

import (
	"encoding/json"
	"errors"
	"lib/marshal"

	"github.com/cloudfoundry-incubator/garden"
	"github.com/cloudfoundry-incubator/garden/fakes"
	"github.com/cloudfoundry-incubator/guardian-cni-adapter/lookup"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Lookup", func() {
	var gardenClient *fakes.FakeClient
	var client *lookup.Client
	var fakeContainer *fakes.FakeContainer
	var properties garden.Properties

	BeforeEach(func() {
		gardenClient = &fakes.FakeClient{}
		client = &lookup.Client{
			GardenClient: gardenClient,
			Marshaler:    marshal.MarshalFunc(json.Marshal),
		}

		fakeContainer = &fakes.FakeContainer{}
		properties = map[string]string{
			"network.some_key":        "some_value",
			"network.some_other_key":  "some_other_value",
			"garden.network.whatever": "do not propagate",
			"some.other.property":     "do not propagate",
		}
		gardenClient.LookupReturns(fakeContainer, nil)
		fakeContainer.PropertiesReturns(properties, nil)

	})

	It("returns a network spec that can be merged into CNI plugins", func() {
		spec, err := client.GetNetworkSpec("some-handle")
		Expect(err).NotTo(HaveOccurred())

		Expect(spec).To(MatchJSON(`{ "some_key": "some_value", "some_other_key": "some_other_value" }`))
	})

	Context("when the garden client lookup fails", func() {
		BeforeEach(func() {
			gardenClient.LookupReturns(nil, errors.New("some error"))
		})
		It("returns a meaningful error", func() {
			_, err := client.GetNetworkSpec("some-handle")
			Expect(err).To(MatchError(`garden lookup container "some-handle": some error`))
		})
	})

	Context("when the container properties fails", func() {
		BeforeEach(func() {
			fakeContainer.PropertiesReturns(nil, errors.New("some error"))
		})
		It("returns a meaningful error", func() {
			_, err := client.GetNetworkSpec("some-handle")
			Expect(err).To(MatchError(`properties for "some-handle": some error`))
		})
	})

	Context("when the container has no network properties", func() {
		BeforeEach(func() {
			delete(properties, "network.some_key")
			delete(properties, "network.some_other_key")
		})
		It("returns a meaningful error", func() {
			spec, err := client.GetNetworkSpec("some-handle")
			Expect(err).NotTo(HaveOccurred())
			Expect(spec).To(Equal(""))
		})
	})

	Context("when marshaling the spec json fails", func() {
		BeforeEach(func() {
			client.Marshaler = marshal.MarshalFunc(func(_ interface{}) ([]byte, error) {
				return nil, errors.New("potato")
			})
		})
		It("returns a meaningful error", func() {
			_, err := client.GetNetworkSpec("some-handle")
			Expect(err).To(MatchError(`marshal spec for "some-handle": potato`))
		})
	})
})
