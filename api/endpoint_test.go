package api_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"code.cloudfoundry.org/cli/v8/plugin/pluginfakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	. "code.cloudfoundry.org/app-autoscaler-cli-plugin/api"
)

var _ = Describe("Endpoint Helper Test", func() {

	const (
		fakeApiEndpoint = "autoscaler.boshlite.com"
	)

	var (
		endpoint       *APIEndpoint
		cfclient       *CFClient
		configFilePath string
		content        []byte
		err            error
		apiServer      *ghttp.Server
		cliConnection  *pluginfakes.FakeCliConnection
	)

	BeforeEach(func() {
		os.Setenv("AUTOSCALER_CONFIG_FILE", "test_config.json")
		configFilePath = ConfigFile()
		cliConnection = &pluginfakes.FakeCliConnection{}
	})

	AfterEach(func() {
		os.RemoveAll("plugins")
	})

	Context("Set API endpoint", func() {

		BeforeEach(func() {
			apiServer = ghttp.NewServer()
			apiServer.RouteToHandler("GET", "/health",
				ghttp.RespondWith(http.StatusOK, ""),
			)

			cliConnection.ApiEndpointReturns(apiServer.URL(), nil)
			cliConnection.IsSSLDisabledReturns(false, nil)
			cfclient, err = NewCFClient(cliConnection)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("When endpoint is valid", func() {
			BeforeEach(func() {
				err = SetEndpoint(cfclient, apiServer.URL()+"/", false)
				Expect(err).NotTo(HaveOccurred())
			})

			It("Set a valid json to config file", func() {
				err = SetEndpoint(cfclient, apiServer.URL(), false)
				Expect(err).NotTo(HaveOccurred())

				content, err = ioutil.ReadFile(configFilePath)
				Expect(err).NotTo(HaveOccurred())
				Expect(content).Should(MatchJSON(fmt.Sprintf(`{"URL":"%s", "SkipSSLValidation":%t}`, apiServer.URL(), false)))
			})

			It("it prune the last /", func() {
				content, err = ioutil.ReadFile(configFilePath)
				Expect(err).NotTo(HaveOccurred())
				Expect(content).Should(MatchJSON(fmt.Sprintf(`{"URL":"%s", "SkipSSLValidation":%t}`, apiServer.URL(), false)))
			})
		})

		Context("When autoscaler domain doesn't match CF API domain", func() {
			BeforeEach(func() {
				cliConnection.ApiEndpointReturns("api.bosh-lite.com", nil)
				cliConnection.IsSSLDisabledReturns(false, nil)
				cfclient, err = NewCFClient(cliConnection)
				Expect(err).NotTo(HaveOccurred())
			})
			It("it fails", func() {
				err = SetEndpoint(cfclient, apiServer.URL(), false)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("When autoscaler endpoint doesn't exist", func() {
			BeforeEach(func() {
				apiServer.RouteToHandler("GET", "/health",
					ghttp.RespondWith(http.StatusNotFound, ""),
				)
			})

			It("it fails", func() {
				err = SetEndpoint(cfclient, apiServer.URL(), false)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Context("Unset API endpoint", func() {

		BeforeEach(func() {
			urlConfig := []byte(fmt.Sprintf(`{"URL":"%s"}`, fakeApiEndpoint))
			err = ioutil.WriteFile(configFilePath, urlConfig, 0600)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Succed and set config.json to an empty file", func() {
			err = UnsetEndpoint()
			Expect(err).NotTo(HaveOccurred())

			content, err = ioutil.ReadFile(configFilePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(content)).Should(Equal(0))
		})
	})

	Context("Get API Endpoint", func() {

		BeforeEach(func() {
			apiServer = ghttp.NewServer()
			apiServer.RouteToHandler("GET", "/health",
				ghttp.RespondWith(http.StatusOK, ""),
			)

			cliConnection.ApiEndpointReturns(apiServer.URL(), nil)
			cliConnection.IsSSLDisabledReturns(false, nil)
			cfclient, err = NewCFClient(cliConnection)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when there is an existing endpoint defined in configuration file", func() {

			BeforeEach(func() {
				urlConfig := []byte(fmt.Sprintf(`{"URL":"%s"}`, apiServer.URL()))
				err = ioutil.WriteFile(configFilePath, urlConfig, 0600)
				Expect(err).NotTo(HaveOccurred())
			})

			It("Return the existing URL when it's domain still consistent with the current cf domain", func() {
				endpoint, err = GetEndpoint(cfclient)
				Expect(err).NotTo(HaveOccurred())
				Expect(endpoint.URL).Should(Equal(apiServer.URL()))
			})

			Context("when cf domain changes", func() {

				BeforeEach(func() {
					urlConfig := []byte(fmt.Sprintf(`{"URL":"%s"}`, "autoscaler.bosh-lite.com"))
					err = ioutil.WriteFile(configFilePath, urlConfig, 0600)
					Expect(err).NotTo(HaveOccurred())
				})

				It("Clear staled setting and return the default autoscaler endpoint if it does work ", func() {
					endpoint, err = GetEndpoint(cfclient)
					Expect(err).NotTo(HaveOccurred())
					Expect(endpoint.URL).Should(Equal(apiServer.URL()))
				})

				Context("When default autoscaler endpoint doesn't exist", func() {
					BeforeEach(func() {
						apiServer.RouteToHandler("GET", "/health",
							ghttp.RespondWith(http.StatusNotFound, ""),
						)
					})

					It("Clear staled setting and set the endpoint to empty", func() {
						endpoint, err = GetEndpoint(cfclient)
						Expect(err).NotTo(HaveOccurred())
						Expect(endpoint.URL).Should(Equal(""))
					})
				})
			})
		})

		Context("when configuration file is empty", func() {

			BeforeEach(func() {
				err = ioutil.WriteFile(configFilePath, nil, 0600)
				Expect(err).NotTo(HaveOccurred())
			})

			It("Return a default URL when it is an valid autoscaler api server", func() {
				endpoint, err = GetEndpoint(cfclient)
				Expect(err).NotTo(HaveOccurred())
				Expect(endpoint.URL).Should(Equal(apiServer.URL()))
			})

			Context("When default autoscaler endpoint doesn't exist", func() {

				BeforeEach(func() {
					apiServer.RouteToHandler("GET", "/health",
						ghttp.RespondWith(http.StatusNotFound, ""),
					)
				})

				It("Return empty string ", func() {
					endpoint, err = GetEndpoint(cfclient)
					Expect(err).NotTo(HaveOccurred())
					Expect(endpoint.URL).Should(Equal(""))
				})
			})
		})

		Context("When configuration file is invalid", func() {

			Context("with an invalidJSON file", func() {

				BeforeEach(func() {
					invalidConfig := []byte(`invalidJSON`)
					err = ioutil.WriteFile(configFilePath, invalidConfig, 0600)
					Expect(err).NotTo(HaveOccurred())
				})

				It("Clear the wrong setting and return the default autoscaler endpoint if it works", func() {
					endpoint, err = GetEndpoint(cfclient)
					Expect(err).NotTo(HaveOccurred())
					Expect(endpoint.URL).Should(Equal(apiServer.URL()))
				})

			})

			Context("when no URL field defined in config file", func() {

				BeforeEach(func() {
					invalidConfig := []byte(`{"invalidJSON": invalidJSON}`)
					err = ioutil.WriteFile(configFilePath, invalidConfig, 0600)
					Expect(err).NotTo(HaveOccurred())
				})

				It("Clear the wrong setting and return the default autoscaler endpoint if it works", func() {
					endpoint, err = GetEndpoint(cfclient)
					Expect(err).NotTo(HaveOccurred())
					Expect(endpoint.URL).Should(Equal(apiServer.URL()))
				})

			})

		})
	})
})
