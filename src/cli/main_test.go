package main_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"cli/api"
	. "cli/models"
	"cli/ui"
	cjson "cli/util/json"

	"code.cloudfoundry.org/cli/plugin/models"
	"code.cloudfoundry.org/cli/util/testhelpers/rpcserver"
	"code.cloudfoundry.org/cli/util/testhelpers/rpcserver/rpcserverfakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/ghttp"
	. "github.com/onsi/gomega/gstruct"
)

var _ = Describe("App-AutoScaler Commands", func() {
	const (
		fakeAppName     string = "fakeAppName"
		fakeAppId       string = "fakeAppId"
		fakeAccessToken string = "fakeAccessToken"
		outputFile      string = "output.txt"
	)

	var (
		rpcHandlers *rpcserverfakes.FakeHandlers
		ts          *rpcserver.TestServer
		apiServer   *ghttp.Server
		apiEndpoint string
		args        []string
		session     *gexec.Session
		err         error

		fakePolicy ScalingPolicy = ScalingPolicy{
			InstanceMin: 1,
			InstanceMax: 2,
			ScalingRules: []*ScalingRule{
				{
					MetricType:            "memoryused",
					StatWindowSeconds:     300,
					BreachDurationSeconds: 600,
					Threshold:             30,
					Operator:              "<=",
					CoolDownSeconds:       300,
					Adjustment:            "-2",
				},
				{
					MetricType:            "memoryused",
					StatWindowSeconds:     300,
					BreachDurationSeconds: 600,
					Threshold:             30,
					Operator:              ">",
					CoolDownSeconds:       300,
					Adjustment:            "+100%",
				},
			},
			Schedules: &ScalingSchedules{
				Timezone: "Asia/Shanghai",
				RecurringSchedules: []*RecurringSchedule{
					{
						StartTime:             "10:00",
						EndTime:               "18:00",
						DaysOfWeek:            []int{1, 2, 3},
						ScheduledInstanceMin:  5,
						ScheduledInstanceMax:  20,
						ScheduledInstanceInit: 10,
					},
				},
				SpecificDateSchedules: []*SpecificDateSchedule{
					{
						StartDateTime:         "2006-01-02T15:04",
						EndDateTime:           "2006-01-02T15:04",
						ScheduledInstanceMin:  10,
						ScheduledInstanceMax:  50,
						ScheduledInstanceInit: 30,
					},
				},
			},
		}
	)

	BeforeEach(func() {

		os.Unsetenv("CF_TRACE")
		os.Setenv("AUTOSCALER_CONFIG_FILE", "test_config.json")

		//start fake AutoScaler API server
		apiServer = ghttp.NewServer()
		apiServer.RouteToHandler("GET", "/health",
			ghttp.RespondWith(http.StatusOK, ""),
		)
		apiEndpoint = apiServer.URL()

		//start rpc server to test cf cli plugin
		rpcHandlers = new(rpcserverfakes.FakeHandlers)

		//set rpc.CallCoreCommand to a successful call
		//rpc.CallCoreCommand is used in both cliConnection.CliCommand() and
		//cliConnection.CliWithoutTerminalOutput()
		rpcHandlers.CallCoreCommandStub = func(_ []string, retVal *bool) error {
			*retVal = true
			return nil
		}
		//set rpc.GetOutputAndReset to return empty string; this is used by CliCommand()/CliWithoutTerminalOutput()
		rpcHandlers.GetOutputAndResetStub = func(_ bool, retVal *[]string) error {
			*retVal = []string{"{}"}
			return nil
		}
		rpcHandlers.ApiEndpointStub = func(_ string, retVal *string) error {
			*retVal = apiEndpoint
			return nil
		}

		ts, err = rpcserver.NewTestRPCServer(rpcHandlers)
		Expect(err).NotTo(HaveOccurred())

		err = ts.Start()
		Expect(err).NotTo(HaveOccurred())

	})

	AfterEach(func() {
		ts.Stop()
		apiServer.Close()
		os.Remove(outputFile)
		os.RemoveAll("plugins")
	})

	Describe("Commands autoscaling-api, asa", func() {

		Context("Set api endpoint", func() {

			BeforeEach(func() {
				args = []string{ts.Port(), "autoscaling-api", apiEndpoint}
			})

			Context("with http server", func() {
				Context("When endpoint url is valid", func() {
					It("Succeed' ", func() {
						session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
						Expect(err).NotTo(HaveOccurred())
						session.Wait()
						Expect(session).To(gbytes.Say(ui.SetAPIEndpoint, apiEndpoint))
						Expect(session.ExitCode()).To(Equal(0))
					})
				})

				Context("When api server is unaccessible ", func() {
					BeforeEach(func() {
						apiServer.Close()
					})

					It("Failed with connection refused", func() {
						session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
						Expect(err).NotTo(HaveOccurred())
						session.Wait()
						Expect(session).To(gbytes.Say("connection refused"))
						Expect(session.ExitCode()).To(Equal(1))
					})
				})

				Context("When no /health endpoint ", func() {

					BeforeEach(func() {
						apiServer.RouteToHandler("GET", "/health",
							ghttp.RespondWith(http.StatusNotFound, ""),
						)
					})

					It("Failed with invalid api endpoint", func() {
						session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
						Expect(err).NotTo(HaveOccurred())
						session.Wait()
						Expect(session).To(gbytes.Say(ui.InvalidAPIEndpoint, apiEndpoint))
						Expect(session.ExitCode()).To(Equal(1))
					})
				})

				Context("When cf api is not set ", func() {

					BeforeEach(func() {
						rpcHandlers.ApiEndpointStub = func(_ string, retVal *string) error {
							*retVal = ""
							return nil
						}
					})

					It("Failed with missing cf api setting", func() {
						session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
						Expect(err).NotTo(HaveOccurred())
						session.Wait()
						Expect(session).To(gbytes.Say(ui.NOCFAPIEndpoint))
						Expect(session.ExitCode()).To(Equal(1))
					})
				})

				Context("When autoscaler api domain is inconsitent with CF domain ", func() {

					BeforeEach(func() {
						rpcHandlers.ApiEndpointStub = func(_ string, retVal *string) error {
							*retVal = "api.bosh-lite.com"
							return nil
						}
					})

					It("Failed with inconsistent domain", func() {
						session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
						Expect(err).NotTo(HaveOccurred())
						session.Wait()
						Expect(session).To(gbytes.Say(ui.InconsistentDomain, apiEndpoint, "api.bosh-lite.com"))
						Expect(session.ExitCode()).To(Equal(1))
					})
				})

			})

			Context("with a self-signed TLS server", func() {
				var (
					apiTLSServer   *ghttp.Server
					apiTLSEndpoint string
				)

				BeforeEach(func() {
					apiTLSServer = ghttp.NewTLSServer()

					apiTLSServer.RouteToHandler("GET", "/health",
						ghttp.RespondWith(http.StatusOK, ""),
					)
					apiTLSEndpoint = apiTLSServer.URL()
				})

				AfterEach(func() {
					apiTLSServer.Close()
				})

				It("require --skip-ssl-validation option", func() {
					args = []string{ts.Port(), "autoscaling-api", apiTLSEndpoint}
					session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()
					Expect(session).To(gbytes.Say(ui.InvalidSSLCerts, apiTLSEndpoint))
					Expect(session.ExitCode()).To(Equal(1))
				})

				It("succeed with --skip-ssl-validation ", func() {
					args = []string{ts.Port(), "autoscaling-api", apiTLSEndpoint, "--skip-ssl-validation"}
					session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()
					Expect(session).To(gbytes.Say(ui.SetAPIEndpoint, apiTLSEndpoint))
					Expect(session.ExitCode()).To(Equal(0))
				})

				It("attach 'https' as the default protocol when prefix is missing ", func() {
					args = []string{ts.Port(), "autoscaling-api", strings.TrimPrefix(apiTLSEndpoint, "https://"), "--skip-ssl-validation"}
					session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()
					Expect(session).To(gbytes.Say(ui.SetAPIEndpoint, apiTLSEndpoint))
					Expect(session.ExitCode()).To(Equal(0))
				})
			})
		})

		Context("Unset api endpoint", func() {

			It("succeed", func() {
				args = []string{ts.Port(), "autoscaling-api", "--unset"}
				session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				session.Wait()
				Expect(session).To(gbytes.Say(ui.UnsetAPIEndpoint))
				Expect(session.ExitCode()).To(Equal(0))
			})

			It("'unset take higher proprity than the other argument", func() {
				args = []string{ts.Port(), "autoscaling-api", apiEndpoint, "--unset"}
				session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				session.Wait()
				Expect(session).To(gbytes.Say(ui.UnsetAPIEndpoint))
				Expect(session.ExitCode()).To(Equal(0))
			})
		})

		Context("Get api endpoint", func() {

			Context("When no previous endpoint setting", func() {

				Context("When config file doesn't exist at all", func() {
					It("Succeed with the default autoscaler endpoint", func() {
						args = []string{ts.Port(), "autoscaling-api"}
						session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
						Expect(err).NotTo(HaveOccurred())
						session.Wait()
						Expect(session).To(gbytes.Say(ui.APIEndpoint, apiEndpoint))
						Expect(session.ExitCode()).To(Equal(0))
					})
				})

				Context("When config file exists with empty content", func() {
					BeforeEach(func() {
						args = []string{ts.Port(), "autoscaling-api", "--unset"}
						session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
						Expect(err).NotTo(HaveOccurred())
						session.Wait()
					})

					It("Succeed with the default autoscaler endpoint", func() {
						args = []string{ts.Port(), "autoscaling-api"}
						session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
						Expect(err).NotTo(HaveOccurred())
						session.Wait()
						Expect(session).To(gbytes.Say(ui.APIEndpoint, apiEndpoint))
						Expect(session.ExitCode()).To(Equal(0))
					})
				})
			})

			Context("When endpoint was set", func() {

				BeforeEach(func() {
					args = []string{ts.Port(), "autoscaling-api", apiEndpoint}
					session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()
				})

				It("Succeed with the pre-defined endpoint..", func() {
					args = []string{ts.Port(), "autoscaling-api"}
					session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()
					Expect(session).To(gbytes.Say(ui.APIEndpoint, apiEndpoint))
					Expect(session.ExitCode()).To(Equal(0))
				})
			})

			Context("When cf api is not set ", func() {
				BeforeEach(func() {

					rpcHandlers.ApiEndpointStub = func(_ string, retVal *string) error {
						*retVal = ""
						return nil
					}

					args = []string{ts.Port(), "autoscaling-api", apiEndpoint}
					session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)

					rpcHandlers.ApiEndpointStub = func(_ string, retVal *string) error {
						*retVal = ""
						return nil
					}
				})

				It("Failed with missing cf api setting", func() {
					args = []string{ts.Port(), "autoscaling-api"}
					session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()
					Expect(session).To(gbytes.Say(ui.NOCFAPIEndpoint))
					Expect(session.ExitCode()).To(Equal(1))
				})
			})

			Context("When cf api is changed ", func() {
				BeforeEach(func() {
					urlConfig := []byte(fmt.Sprintf(`{"URL":"%s"}`, "autoscaler.bosh-lite.com"))
					err = ioutil.WriteFile(api.ConfigFile(), urlConfig, 0600)
					Expect(err).NotTo(HaveOccurred())

				})

				Context("When the default endpoint works", func() {
					It("succeed to return the default endpoint according to the new domain", func() {
						args = []string{ts.Port(), "autoscaling-api"}
						session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
						Expect(err).NotTo(HaveOccurred())
						session.Wait()
						Expect(session).To(gbytes.Say(ui.APIEndpoint, apiEndpoint))
						Expect(session.ExitCode()).To(Equal(0))
					})
				})

				Context("When the default endpoint doesn't work", func() {

					BeforeEach(func() {
						apiServer.RouteToHandler("GET", "/health",
							ghttp.RespondWith(http.StatusNotFound, ""),
						)
					})

					It("Failed with no api endpoint setting", func() {
						args = []string{ts.Port(), "autoscaling-api"}
						session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
						Expect(err).NotTo(HaveOccurred())
						session.Wait()
						Expect(session).To(gbytes.Say(ui.NoEndpoint))
						Expect(session.ExitCode()).To(Equal(0))
					})
				})
			})

		})

	})

	Describe("Commands autoscaling-policy, asp", func() {

		var urlpath = "/v1/apps/" + fakeAppId + "/policy"
		Context("autoscaling-policy", func() {

			Context("when the args are not properly provided", func() {
				It("Require APP_NAME as argument", func() {
					args = []string{ts.Port(), "autoscaling-policy"}
					session, err := gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()

					Expect(session).To(gbytes.Say("required argument `APP_NAME` was not provided"))
					Expect(session.ExitCode()).To(Equal(1))
				})

				It("Failed when output file path is invalid", func() {
					args = []string{ts.Port(), "autoscaling-policy", fakeAppName, "--output", "invalidDir/invalidFile"}
					session, err := gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()

					Expect(session).To(gbytes.Say("open invalidDir/invalidFile: no such file or directory"))
					Expect(session.ExitCode()).To(Equal(1))
				})
			})

			Context("When cf api is not set ", func() {
				BeforeEach(func() {

					rpcHandlers.ApiEndpointStub = func(_ string, retVal *string) error {
						*retVal = ""
						return nil
					}

					args = []string{ts.Port(), "autoscaling-api", apiEndpoint}
					session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)

					rpcHandlers.ApiEndpointStub = func(_ string, retVal *string) error {
						*retVal = ""
						return nil
					}
				})

				It("Failed with missing cf api setting", func() {
					args = []string{ts.Port(), "autoscaling-policy", fakeAppName}
					session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()
					Expect(session).To(gbytes.Say(ui.NOCFAPIEndpoint))
					Expect(session.ExitCode()).To(Equal(1))
				})
			})

			Context("When cf api is changed ", func() {
				BeforeEach(func() {
					urlConfig := []byte(fmt.Sprintf(`{"URL":"%s"}`, "autoscaler.bosh-lite.com"))
					err = ioutil.WriteFile(api.ConfigFile(), urlConfig, 0600)
					Expect(err).NotTo(HaveOccurred())

				})

				Context("When the default endpoint doesn't work", func() {

					BeforeEach(func() {
						apiServer.RouteToHandler("GET", "/health",
							ghttp.RespondWith(http.StatusNotFound, ""),
						)
					})

					It("Failed with no api endpoint setting", func() {
						args = []string{ts.Port(), "autoscaling-policy", fakeAppName}
						session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
						Expect(err).NotTo(HaveOccurred())
						session.Wait()
						Expect(session).To(gbytes.Say(ui.NoEndpoint))
						Expect(session.ExitCode()).To(Equal(1))
					})
				})
			})

			Context("when cf not login", func() {
				It("exits with 'You must be logged in' error ", func() {
					args = []string{ts.Port(), "autoscaling-policy", fakeAppName}
					session, err := gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()
					Expect(session).To(gbytes.Say("You must be logged in"))
					Expect(session.ExitCode()).To(Equal(1))
				})
			})

			Context("when cf login", func() {
				BeforeEach(func() {
					rpcHandlers.IsLoggedInStub = func(args string, retVal *bool) error {
						*retVal = true
						return nil
					}
				})

				Context("when app not found", func() {
					BeforeEach(func() {
						rpcHandlers.GetAppStub = func(_ string, retVal *plugin_models.GetAppModel) error {
							return errors.New("App fakeApp not found")
						}
					})

					It("exits with 'App not found' error ", func() {
						args = []string{ts.Port(), "autoscaling-policy", fakeAppName}
						session, err := gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
						Expect(err).NotTo(HaveOccurred())
						session.Wait()
						Expect(session).To(gbytes.Say("App fakeApp not found"))
						Expect(session.ExitCode()).To(Equal(1))
					})
				})

				Context("when the app is found", func() {
					BeforeEach(func() {
						rpcHandlers.GetAppStub = func(_ string, retVal *plugin_models.GetAppModel) error {
							*retVal = plugin_models.GetAppModel{
								Guid: fakeAppId,
							}
							return nil
						}
					})

					JustBeforeEach(func() {
						args = []string{ts.Port(), "autoscaling-api", apiEndpoint}
						session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
						Expect(err).NotTo(HaveOccurred())
						session.Wait()
					})

					Context("when access token is wrong", func() {
						BeforeEach(func() {
							rpcHandlers.AccessTokenStub = func(args string, retVal *string) error {
								*retVal = "incorrectAccessToken"
								return nil
							}

							apiServer.RouteToHandler("GET", urlpath,
								ghttp.RespondWith(http.StatusUnauthorized, ""),
							)
						})

						It("failed with 401 error", func() {
							args = []string{ts.Port(), "autoscaling-policy", fakeAppName}
							session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
							Expect(err).NotTo(HaveOccurred())
							session.Wait()

							Expect(session).To(gbytes.Say("Failed to access AutoScaler API endpoint"))
							Expect(session.ExitCode()).To(Equal(1))
						})
					})

					Context("when access token is correct", func() {
						BeforeEach(func() {
							rpcHandlers.AccessTokenStub = func(args string, retVal *string) error {
								*retVal = fakeAccessToken
								return nil
							}
						})

						Context("when policy not found", func() {
							BeforeEach(func() {
								apiServer.RouteToHandler("GET", urlpath,
									ghttp.RespondWith(http.StatusNotFound, ""),
								)
							})

							It("404 returned", func() {
								args = []string{ts.Port(), "autoscaling-policy", fakeAppName}
								session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
								Expect(err).NotTo(HaveOccurred())
								session.Wait()

								Expect(session).To(gbytes.Say(ui.PolicyNotFound, fakeAppName))
								Expect(session.ExitCode()).To(Equal(1))

							})
						})

						Context("when policy exist ", func() {
							BeforeEach(func() {
								apiServer.RouteToHandler("GET", urlpath,
									ghttp.CombineHandlers(
										ghttp.RespondWithJSONEncoded(http.StatusOK, &fakePolicy),
										ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
									),
								)
							})

							It("Succeed to print the policy to stdout", func() {

								args = []string{ts.Port(), "autoscaling-policy", fakeAppName}
								session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
								Expect(err).NotTo(HaveOccurred())
								session.Wait()

								Expect(session.Out).To(gbytes.Say(ui.ShowPolicyHint, fakeAppName))
								policy := bytes.TrimPrefix(session.Out.Contents(), []byte(fmt.Sprintf(ui.ShowPolicyHint+"\n", fakeAppName)))

								var actualPolicy ScalingPolicy
								_ = json.Unmarshal(policy, &actualPolicy)

								Expect(actualPolicy).To(MatchFields(IgnoreExtras, Fields{
									"InstanceMin": BeNumerically("==", fakePolicy.InstanceMin),
									"InstanceMax": BeNumerically("==", fakePolicy.InstanceMax),
								}))

								Expect(*actualPolicy.ScalingRules[0]).To(MatchFields(IgnoreExtras, Fields{
									"MetricType":            Equal(fakePolicy.ScalingRules[0].MetricType),
									"StatWindowSeconds":     BeNumerically("==", fakePolicy.ScalingRules[0].StatWindowSeconds),
									"BreachDurationSeconds": BeNumerically("==", fakePolicy.ScalingRules[0].BreachDurationSeconds),
									"Threshold":             BeNumerically("==", fakePolicy.ScalingRules[0].Threshold),
									"Operator":              Equal(fakePolicy.ScalingRules[0].Operator),
									"CoolDownSeconds":       BeNumerically("==", fakePolicy.ScalingRules[0].CoolDownSeconds),
									"Adjustment":            Equal(fakePolicy.ScalingRules[0].Adjustment),
								}))

								Expect(*actualPolicy.ScalingRules[1]).To(MatchFields(IgnoreExtras, Fields{
									"MetricType":            Equal(fakePolicy.ScalingRules[1].MetricType),
									"StatWindowSeconds":     BeNumerically("==", fakePolicy.ScalingRules[1].StatWindowSeconds),
									"BreachDurationSeconds": BeNumerically("==", fakePolicy.ScalingRules[1].BreachDurationSeconds),
									"Threshold":             BeNumerically("==", fakePolicy.ScalingRules[1].Threshold),
									"Operator":              Equal(fakePolicy.ScalingRules[1].Operator),
									"CoolDownSeconds":       BeNumerically("==", fakePolicy.ScalingRules[1].CoolDownSeconds),
									"Adjustment":            Equal(fakePolicy.ScalingRules[1].Adjustment),
								}))

								Expect(*actualPolicy.Schedules).To(MatchFields(IgnoreExtras, Fields{
									"Timezone": Equal(fakePolicy.Schedules.Timezone),
								}))

								Expect(*actualPolicy.Schedules.RecurringSchedules[0]).To(MatchFields(IgnoreExtras, Fields{
									"StartTime":             Equal(fakePolicy.Schedules.RecurringSchedules[0].StartTime),
									"EndTime":               Equal(fakePolicy.Schedules.RecurringSchedules[0].EndTime),
									"DaysOfWeek":            Equal(fakePolicy.Schedules.RecurringSchedules[0].DaysOfWeek),
									"ScheduledInstanceMin":  Equal(fakePolicy.Schedules.RecurringSchedules[0].ScheduledInstanceMin),
									"ScheduledInstanceMax":  Equal(fakePolicy.Schedules.RecurringSchedules[0].ScheduledInstanceMax),
									"ScheduledInstanceInit": Equal(fakePolicy.Schedules.RecurringSchedules[0].ScheduledInstanceInit),
								}))

								Expect(*actualPolicy.Schedules.SpecificDateSchedules[0]).To(MatchFields(IgnoreExtras, Fields{
									"StartDateTime":         Equal(fakePolicy.Schedules.SpecificDateSchedules[0].StartDateTime),
									"EndDateTime":           Equal(fakePolicy.Schedules.SpecificDateSchedules[0].EndDateTime),
									"ScheduledInstanceMin":  Equal(fakePolicy.Schedules.SpecificDateSchedules[0].ScheduledInstanceMin),
									"ScheduledInstanceMax":  Equal(fakePolicy.Schedules.SpecificDateSchedules[0].ScheduledInstanceMax),
									"ScheduledInstanceInit": Equal(fakePolicy.Schedules.SpecificDateSchedules[0].ScheduledInstanceInit),
								}))

								Expect(session.ExitCode()).To(Equal(0))

							})

							Context("Succeed to print the policy to file", func() {

								It("succeed", func() {
									args = []string{ts.Port(), "autoscaling-policy", fakeAppName, "--output", outputFile}
									session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
									Expect(err).NotTo(HaveOccurred())
									session.Wait()

									Expect(session.Out).To(gbytes.Say(ui.SavePolicyHint, fakeAppName, outputFile))

									Expect(outputFile).To(BeARegularFile())
									contents, err := ioutil.ReadFile(outputFile)
									Expect(err).NotTo(HaveOccurred())

									var actualPolicy ScalingPolicy
									_ = json.Unmarshal(contents, &actualPolicy)

									Expect(actualPolicy).To(MatchFields(IgnoreExtras, Fields{
										"InstanceMin": BeNumerically("==", fakePolicy.InstanceMin),
										"InstanceMax": BeNumerically("==", fakePolicy.InstanceMax),
									}))

									Expect(*actualPolicy.ScalingRules[0]).To(MatchFields(IgnoreExtras, Fields{
										"MetricType":            Equal(fakePolicy.ScalingRules[0].MetricType),
										"StatWindowSeconds":     BeNumerically("==", fakePolicy.ScalingRules[0].StatWindowSeconds),
										"BreachDurationSeconds": BeNumerically("==", fakePolicy.ScalingRules[0].BreachDurationSeconds),
										"Threshold":             BeNumerically("==", fakePolicy.ScalingRules[0].Threshold),
										"Operator":              Equal(fakePolicy.ScalingRules[0].Operator),
										"CoolDownSeconds":       BeNumerically("==", fakePolicy.ScalingRules[0].CoolDownSeconds),
										"Adjustment":            Equal(fakePolicy.ScalingRules[0].Adjustment),
									}))
									Expect(*actualPolicy.ScalingRules[1]).To(MatchFields(IgnoreExtras, Fields{
										"MetricType":            Equal(fakePolicy.ScalingRules[1].MetricType),
										"StatWindowSeconds":     BeNumerically("==", fakePolicy.ScalingRules[1].StatWindowSeconds),
										"BreachDurationSeconds": BeNumerically("==", fakePolicy.ScalingRules[1].BreachDurationSeconds),
										"Threshold":             BeNumerically("==", fakePolicy.ScalingRules[1].Threshold),
										"Operator":              Equal(fakePolicy.ScalingRules[1].Operator),
										"CoolDownSeconds":       BeNumerically("==", fakePolicy.ScalingRules[1].CoolDownSeconds),
										"Adjustment":            Equal(fakePolicy.ScalingRules[1].Adjustment),
									}))

									Expect(*actualPolicy.Schedules).To(MatchFields(IgnoreExtras, Fields{
										"Timezone": Equal(fakePolicy.Schedules.Timezone),
									}))

									Expect(*actualPolicy.Schedules.RecurringSchedules[0]).To(MatchFields(IgnoreExtras, Fields{
										"StartTime":             Equal(fakePolicy.Schedules.RecurringSchedules[0].StartTime),
										"EndTime":               Equal(fakePolicy.Schedules.RecurringSchedules[0].EndTime),
										"DaysOfWeek":            Equal(fakePolicy.Schedules.RecurringSchedules[0].DaysOfWeek),
										"ScheduledInstanceMin":  Equal(fakePolicy.Schedules.RecurringSchedules[0].ScheduledInstanceMin),
										"ScheduledInstanceMax":  Equal(fakePolicy.Schedules.RecurringSchedules[0].ScheduledInstanceMax),
										"ScheduledInstanceInit": Equal(fakePolicy.Schedules.RecurringSchedules[0].ScheduledInstanceInit),
									}))

									Expect(*actualPolicy.Schedules.SpecificDateSchedules[0]).To(MatchFields(IgnoreExtras, Fields{
										"StartDateTime":         Equal(fakePolicy.Schedules.SpecificDateSchedules[0].StartDateTime),
										"EndDateTime":           Equal(fakePolicy.Schedules.SpecificDateSchedules[0].EndDateTime),
										"ScheduledInstanceMin":  Equal(fakePolicy.Schedules.SpecificDateSchedules[0].ScheduledInstanceMin),
										"ScheduledInstanceMax":  Equal(fakePolicy.Schedules.SpecificDateSchedules[0].ScheduledInstanceMax),
										"ScheduledInstanceInit": Equal(fakePolicy.Schedules.SpecificDateSchedules[0].ScheduledInstanceInit),
									}))
									Expect(session.ExitCode()).To(Equal(0))

								})

							})
						})

					})

				})

			})
		})
	})

	Describe("Commands attach-autoscaling-policy, aasp", func() {

		var urlpath = "/v1/apps/" + fakeAppId + "/policy"
		Context("attach-autoscaling-policy", func() {

			Context("when the args are not properly provided", func() {
				It("Require both APP_NAME and PATH_TO_POLICY_FILE as argument", func() {
					args = []string{ts.Port(), "attach-autoscaling-policy"}
					session, err := gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()

					Expect(session).To(gbytes.Say("the required arguments `APP_NAME` and `PATH_TO_POLICY_FILE` were not provided"))
					Expect(session.ExitCode()).To(Equal(1))
				})

				It("Require PATH_TO_POLICY_FILE as argument", func() {
					args = []string{ts.Port(), "attach-autoscaling-policy", fakeAppName}
					session, err := gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()

					Expect(session).To(gbytes.Say("the required argument `PATH_TO_POLICY_FILE` was not provided"))
					Expect(session.ExitCode()).To(Equal(1))
				})
			})

			Context("When cf api is not set ", func() {
				BeforeEach(func() {

					rpcHandlers.ApiEndpointStub = func(_ string, retVal *string) error {
						*retVal = ""
						return nil
					}

					args = []string{ts.Port(), "autoscaling-api", apiEndpoint}
					session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)

					rpcHandlers.ApiEndpointStub = func(_ string, retVal *string) error {
						*retVal = ""
						return nil
					}
				})

				It("Failed with missing cf api setting", func() {
					args = []string{ts.Port(), "attach-autoscaling-policy", fakeAppName, outputFile}
					session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()
					Expect(session).To(gbytes.Say(ui.NOCFAPIEndpoint))
					Expect(session.ExitCode()).To(Equal(1))
				})
			})

			Context("When cf api is changed ", func() {
				BeforeEach(func() {
					urlConfig := []byte(fmt.Sprintf(`{"URL":"%s"}`, "autoscaler.bosh-lite.com"))
					err = ioutil.WriteFile(api.ConfigFile(), urlConfig, 0600)
					Expect(err).NotTo(HaveOccurred())

				})

				Context("When the default endpoint doesn't work", func() {

					BeforeEach(func() {
						apiServer.RouteToHandler("GET", "/health",
							ghttp.RespondWith(http.StatusNotFound, ""),
						)
					})

					It("Failed with no api endpoint setting", func() {
						args = []string{ts.Port(), "attach-autoscaling-policy", fakeAppName, outputFile}
						session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
						Expect(err).NotTo(HaveOccurred())
						session.Wait()
						Expect(session).To(gbytes.Say(ui.NoEndpoint))
						Expect(session.ExitCode()).To(Equal(1))
					})
				})
			})

			Context("when cf not login", func() {
				It("exits with 'You must be logged in' error ", func() {
					args = []string{ts.Port(), "attach-autoscaling-policy", fakeAppName, outputFile}
					session, err := gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()
					Expect(session).To(gbytes.Say("You must be logged in"))
					Expect(session.ExitCode()).To(Equal(1))
				})
			})

			Context("when cf login", func() {
				BeforeEach(func() {
					rpcHandlers.IsLoggedInStub = func(args string, retVal *bool) error {
						*retVal = true
						return nil
					}
				})

				Context("when app not found", func() {
					BeforeEach(func() {
						rpcHandlers.GetAppStub = func(_ string, retVal *plugin_models.GetAppModel) error {
							return errors.New("App fakeApp not found")
						}
					})

					It("exits with 'App not found' error ", func() {
						args = []string{ts.Port(), "attach-autoscaling-policy", fakeAppName, outputFile}
						session, err := gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
						Expect(err).NotTo(HaveOccurred())
						session.Wait()
						Expect(session).To(gbytes.Say("App fakeApp not found"))
						Expect(session.ExitCode()).To(Equal(1))
					})
				})

				Context("when the app is found", func() {
					BeforeEach(func() {
						rpcHandlers.GetAppStub = func(_ string, retVal *plugin_models.GetAppModel) error {
							*retVal = plugin_models.GetAppModel{
								Guid: fakeAppId,
							}
							return nil
						}
					})

					JustBeforeEach(func() {
						args = []string{ts.Port(), "autoscaling-api", apiEndpoint}
						session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
						Expect(err).NotTo(HaveOccurred())
						session.Wait()
					})

					Context("when policy file is not exist", func() {
						BeforeEach(func() {
							err = os.Remove(outputFile)
						})

						It("Failed when policy file not exist", func() {
							args = []string{ts.Port(), "attach-autoscaling-policy", fakeAppName, outputFile}
							session, err := gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
							Expect(err).NotTo(HaveOccurred())
							session.Wait()
							Expect(session).To(gbytes.Say(ui.FailToLoadPolicyFile, outputFile))
							Expect(session.ExitCode()).To(Equal(1))
						})
					})

					Context("when policy file is empty", func() {
						BeforeEach(func() {
							err = ioutil.WriteFile(outputFile, nil, 0666)
							Expect(err).NotTo(HaveOccurred())
						})

						It("Failed when policy file is empty", func() {
							args = []string{ts.Port(), "attach-autoscaling-policy", fakeAppName, outputFile}
							session, err := gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
							Expect(err).NotTo(HaveOccurred())
							session.Wait()
							Expect(session).To(gbytes.Say(strings.TrimSuffix(ui.InvalidPolicy, "%v.")))
							Expect(session.ExitCode()).To(Equal(1))
						})
					})

					Context("when policy file is invalid json", func() {
						BeforeEach(func() {
							invalidPolicy := []byte(`{"policy":invalidPolicy}`)
							err = ioutil.WriteFile(outputFile, invalidPolicy, 0666)
							Expect(err).NotTo(HaveOccurred())
						})

						It("Failed when policy file is empty", func() {
							args = []string{ts.Port(), "attach-autoscaling-policy", fakeAppName, outputFile}
							session, err := gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
							Expect(err).NotTo(HaveOccurred())
							session.Wait()
							Expect(session).To(gbytes.Say(strings.TrimSuffix(ui.InvalidPolicy, "%v.")))
							Expect(session.ExitCode()).To(Equal(1))
						})
					})

					Context("when both app & policy is written in json format correctly", func() {

						BeforeEach(func() {
							policyBytes, err := cjson.MarshalWithoutHTMLEscape(fakePolicy)
							Expect(err).NotTo(HaveOccurred())
							err = ioutil.WriteFile(outputFile, policyBytes, 0666)
							Expect(err).NotTo(HaveOccurred())
						})

						Context("when access token is wrong", func() {
							BeforeEach(func() {
								rpcHandlers.AccessTokenStub = func(args string, retVal *string) error {
									*retVal = "incorrectAccessToken"
									return nil
								}

								apiServer.RouteToHandler("PUT", urlpath,
									ghttp.RespondWith(http.StatusUnauthorized, ""),
								)
							})

							It("failed with 401 error", func() {
								args = []string{ts.Port(), "attach-autoscaling-policy", fakeAppName, outputFile}
								session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
								Expect(err).NotTo(HaveOccurred())
								session.Wait()

								Expect(session).To(gbytes.Say("Failed to access AutoScaler API endpoint"))
								Expect(session.ExitCode()).To(Equal(1))
							})
						})

						Context("when access token is correct", func() {
							BeforeEach(func() {
								rpcHandlers.AccessTokenStub = func(args string, retVal *string) error {
									*retVal = fakeAccessToken
									return nil
								}
							})

							Context("when attached policy definition is invalid ", func() {
								BeforeEach(func() {
									apiServer.RouteToHandler("PUT", urlpath,
										ghttp.CombineHandlers(
											ghttp.RespondWith(http.StatusBadRequest, `{"success":false,"error":[{"property":"instance_min_count","message":"instance_min_count and instance_max_count values are not compatible","instance":{"instance_max_count":2,"instance_min_count":10,"scaling_rules":[{"adjustment":"+1","breach_duration_secs":600,"cool_down_secs":300,"metric_type":"memoryused","operator":">","stat_window_secs":300,"threshold":100},{"adjustment":"-1","breach_duration_secs":600,"cool_down_secs":300,"metric_type":"memoryused","operator":"<=","stat_window_secs":300,"threshold":5}]},"stack":"instance_min_count 10 is higher or equal to instance_max_count 2 in policy_json"}],"result":null}`),
											ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
										),
									)

									fakePolicy.InstanceMin = 10
									fakePolicy.InstanceMax = 2
									policyBytes, err := cjson.MarshalWithoutHTMLEscape(fakePolicy)
									Expect(err).NotTo(HaveOccurred())
									err = ioutil.WriteFile(outputFile, policyBytes, 0666)
									Expect(err).NotTo(HaveOccurred())

								})

								It("Failed with 400", func() {

									args = []string{ts.Port(), "attach-autoscaling-policy", fakeAppName, outputFile}
									session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
									Expect(err).NotTo(HaveOccurred())
									session.Wait()

									Expect(session.Out).To(gbytes.Say(ui.AttachPolicyHint, fakeAppName))
									Expect(session).To(gbytes.Say("FAILED"))
									Expect(session).To(gbytes.Say(ui.InvalidPolicy, "\n"+"instance_min_count 10 is higher or equal to instance_max_count 2 in policy_json"))
									Expect(session.ExitCode()).To(Equal(1))

								})
							})

							Context("when No policy defined previously", func() {
								BeforeEach(func() {
									apiServer.RouteToHandler("PUT", urlpath,
										ghttp.CombineHandlers(
											ghttp.RespondWith(http.StatusCreated, ""),
											ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
										),
									)
								})

								It("Succeed with 201", func() {
									args = []string{ts.Port(), "attach-autoscaling-policy", fakeAppName, outputFile}
									session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
									Expect(err).NotTo(HaveOccurred())
									session.Wait()

									Expect(session.Out).To(gbytes.Say(ui.AttachPolicyHint, fakeAppName))
									Expect(session.Out).To(gbytes.Say("OK"))
									Expect(session.ExitCode()).To(Equal(0))

								})
							})

							Context("when policy exist previously ", func() {
								BeforeEach(func() {
									apiServer.RouteToHandler("PUT", urlpath,
										ghttp.CombineHandlers(
											ghttp.RespondWith(http.StatusOK, ""),
											ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
										),
									)
								})

								It("Succeed with 200", func() {

									args = []string{ts.Port(), "attach-autoscaling-policy", fakeAppName, outputFile}
									session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
									Expect(err).NotTo(HaveOccurred())
									session.Wait()

									Expect(session.Out).To(gbytes.Say(ui.AttachPolicyHint, fakeAppName))
									Expect(session.Out).To(gbytes.Say("OK"))
									Expect(session.ExitCode()).To(Equal(0))

								})

							})

						})
					})

				})

			})
		})
	})

	Describe("Commands detach-autoscaling-policy, dasp", func() {

		var urlpath = "/v1/apps/" + fakeAppId + "/policy"
		Context("detach-autoscaling-policy", func() {

			Context("when the args are not properly provided", func() {
				It("Require APP_NAME as argument", func() {
					args = []string{ts.Port(), "detach-autoscaling-policy"}
					session, err := gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()

					Expect(session).To(gbytes.Say("required argument `APP_NAME` was not provided"))
					Expect(session.ExitCode()).To(Equal(1))
				})
			})

			Context("When cf api is not set ", func() {
				BeforeEach(func() {

					rpcHandlers.ApiEndpointStub = func(_ string, retVal *string) error {
						*retVal = ""
						return nil
					}

					args = []string{ts.Port(), "autoscaling-api", apiEndpoint}
					session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)

					rpcHandlers.ApiEndpointStub = func(_ string, retVal *string) error {
						*retVal = ""
						return nil
					}
				})

				It("Failed with missing cf api setting", func() {
					args = []string{ts.Port(), "detach-autoscaling-policy", fakeAppName}
					session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()
					Expect(session).To(gbytes.Say(ui.NOCFAPIEndpoint))
					Expect(session.ExitCode()).To(Equal(1))
				})
			})

			Context("When cf api is changed ", func() {
				BeforeEach(func() {
					urlConfig := []byte(fmt.Sprintf(`{"URL":"%s"}`, "autoscaler.bosh-lite.com"))
					err = ioutil.WriteFile(api.ConfigFile(), urlConfig, 0600)
					Expect(err).NotTo(HaveOccurred())

				})

				Context("When the default endpoint doesn't work", func() {

					BeforeEach(func() {
						apiServer.RouteToHandler("GET", "/health",
							ghttp.RespondWith(http.StatusNotFound, ""),
						)
					})

					It("Failed with no api endpoint setting", func() {
						args = []string{ts.Port(), "detach-autoscaling-policy", fakeAppName}
						session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
						Expect(err).NotTo(HaveOccurred())
						session.Wait()
						Expect(session).To(gbytes.Say(ui.NoEndpoint))
						Expect(session.ExitCode()).To(Equal(1))
					})
				})
			})

			Context("when cf not login", func() {
				It("exits with 'You must be logged in' error ", func() {
					args = []string{ts.Port(), "detach-autoscaling-policy", fakeAppName}
					session, err := gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()
					Expect(session).To(gbytes.Say("You must be logged in"))
					Expect(session.ExitCode()).To(Equal(1))
				})
			})

			Context("when cf login", func() {
				BeforeEach(func() {
					rpcHandlers.IsLoggedInStub = func(args string, retVal *bool) error {
						*retVal = true
						return nil
					}
				})

				Context("when app not found", func() {
					BeforeEach(func() {
						rpcHandlers.GetAppStub = func(_ string, retVal *plugin_models.GetAppModel) error {
							return errors.New("App fakeApp not found")
						}
					})

					It("exits with 'App not found' error ", func() {
						args = []string{ts.Port(), "detach-autoscaling-policy", fakeAppName}
						session, err := gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
						Expect(err).NotTo(HaveOccurred())
						session.Wait()
						Expect(session).To(gbytes.Say("App fakeApp not found"))
						Expect(session.ExitCode()).To(Equal(1))
					})
				})

				Context("when the app is found", func() {
					BeforeEach(func() {
						rpcHandlers.GetAppStub = func(_ string, retVal *plugin_models.GetAppModel) error {
							*retVal = plugin_models.GetAppModel{
								Guid: fakeAppId,
							}
							return nil
						}
					})

					JustBeforeEach(func() {
						args = []string{ts.Port(), "autoscaling-api", apiEndpoint}
						session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
						Expect(err).NotTo(HaveOccurred())
						session.Wait()
					})

					Context("when access token is wrong", func() {
						BeforeEach(func() {
							rpcHandlers.AccessTokenStub = func(args string, retVal *string) error {
								*retVal = "incorrectAccessToken"
								return nil
							}

							apiServer.RouteToHandler("DELETE", urlpath,
								ghttp.RespondWith(http.StatusUnauthorized, ""),
							)
						})

						It("failed with 401 error", func() {
							args = []string{ts.Port(), "detach-autoscaling-policy", fakeAppName}
							session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
							Expect(err).NotTo(HaveOccurred())
							session.Wait()

							Expect(session).To(gbytes.Say("Failed to access AutoScaler API endpoint"))
							Expect(session.ExitCode()).To(Equal(1))
						})
					})

					Context("when access token is correct", func() {
						BeforeEach(func() {
							rpcHandlers.AccessTokenStub = func(args string, retVal *string) error {
								*retVal = fakeAccessToken
								return nil
							}
						})

						Context("when policy not found", func() {
							BeforeEach(func() {
								apiServer.RouteToHandler("DELETE", urlpath,
									ghttp.RespondWith(http.StatusNotFound, ""),
								)
							})

							It("404 returned", func() {
								args = []string{ts.Port(), "detach-autoscaling-policy", fakeAppName}
								session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
								Expect(err).NotTo(HaveOccurred())
								session.Wait()

								Expect(session.Out).To(gbytes.Say(ui.DetachPolicyHint, fakeAppName))
								Expect(session).To(gbytes.Say(ui.PolicyNotFound, fakeAppName))
								Expect(session.ExitCode()).To(Equal(1))

							})
						})

						Context("when policy exist ", func() {
							BeforeEach(func() {
								apiServer.RouteToHandler("DELETE", urlpath,
									ghttp.CombineHandlers(
										ghttp.RespondWith(http.StatusOK, ""),
										ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
									),
								)
							})

							It("succeed", func() {

								args = []string{ts.Port(), "detach-autoscaling-policy", fakeAppName}
								session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
								Expect(err).NotTo(HaveOccurred())
								session.Wait()

								Expect(session.Out).To(gbytes.Say(ui.DetachPolicyHint, fakeAppName))
								Expect(session.Out).To(gbytes.Say("OK"))
								Expect(session.ExitCode()).To(Equal(0))
							})

						})

					})

				})

			})
		})
	})

	Describe("Commands autoscaling-metrics, asm", func() {

		var (
			metricName               = "memoryused"
			aggregatedMetricsUrlPath = "/v1/apps/" + fakeAppId + "/aggregated_metric_histories/" + metricName
			now                      = time.Now()
			lowPrecisionNowInNano    = (now.UnixNano() / 1E9) * 1E9
		)

		Context("autoscaling-metrics", func() {

			Context("when the args or options are not properly provided", func() {

				It("Require APP_NAME as argument", func() {
					args = []string{ts.Port(), "autoscaling-metrics"}
					session, err := gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()

					Expect(session).To(gbytes.Say("required arguments `APP_NAME` and `METRIC_NAME` were not provided"))
					Expect(session.ExitCode()).To(Equal(1))
				})

				It("Require METRIC_NAME as argument", func() {
					args = []string{ts.Port(), "autoscaling-metrics", fakeAppName}
					session, err := gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()

					Expect(session).To(gbytes.Say("required argument `METRIC_NAME` was not provided"))
					Expect(session.ExitCode()).To(Equal(1))
				})

				It("Failed when METRIC_NAME is unsupported", func() {
					args = []string{ts.Port(), "autoscaling-metrics", fakeAppName, "fakeMetricName"}
					session, err := gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()

					Expect(session).To(gbytes.Say(fmt.Sprintf(ui.UnrecognizedMetricName, "fakeMetricName")))
					Expect(session.ExitCode()).To(Equal(1))
				})

				It("Failed when start/end time is defined in unsupported time format", func() {
					invalidTime := now.Format(time.UnixDate)
					args = []string{ts.Port(), "autoscaling-metrics", fakeAppName, metricName, "--start", invalidTime}
					session, err := gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()

					Expect(session).To(gbytes.Say("Unrecognized date time format"))
					Expect(session.ExitCode()).To(Equal(1))

					args = []string{ts.Port(), "autoscaling-metrics", fakeAppName, metricName, "--end", invalidTime}
					session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()

					Expect(session).To(gbytes.Say("Unrecognized date time format"))
					Expect(session.ExitCode()).To(Equal(1))
				})

				It("Failed when start/end time is prior to 1970-01-01T00:00:00Z", func() {
					args = []string{ts.Port(), "autoscaling-metrics", fakeAppName, metricName,
						"--start", "1969-12-31-T00:00:00Z",
						"--end", "1969-12-31-T23:59:59Z",
					}
					session, err := gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()

					Expect(session).To(gbytes.Say("Unrecognized date time format"))
					Expect(session.ExitCode()).To(Equal(1))
				})

				It("Failed when start time is greater than end time", func() {
					args = []string{ts.Port(), "autoscaling-metrics", fakeAppName, metricName,
						"--start", now.Format(time.RFC3339),
						"--end", time.Unix(0, now.UnixNano()-int64(30*1E9)).Format(time.RFC3339),
					}
					session, err := gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()

					expects := strings.Split(ui.InvalidTimeRange, "%s")
					for _, expect := range expects {
						Expect(session).To(gbytes.Say(expect))
					}
					Expect(session.ExitCode()).To(Equal(1))
				})

				It("Failed when --desc is wrong spelled", func() {
					args = []string{ts.Port(), "autoscaling-metrics", fakeAppName, metricName, "--dddesc"}
					session, err := gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()

					Expect(session).To(gbytes.Say("unknown flag"))
					Expect(session.ExitCode()).To(Equal(1))
				})

				It("Failed when output file path is invalid", func() {
					args = []string{ts.Port(), "autoscaling-metrics", fakeAppName, metricName, "--output", "invalidDir/invalidFile"}
					session, err := gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()

					Expect(session).To(gbytes.Say("open invalidDir/invalidFile: no such file or directory"))
					Expect(session.ExitCode()).To(Equal(1))
				})

			})

			Context("When cf api is not set ", func() {
				BeforeEach(func() {

					rpcHandlers.ApiEndpointStub = func(_ string, retVal *string) error {
						*retVal = ""
						return nil
					}

					args = []string{ts.Port(), "autoscaling-api", apiEndpoint}
					session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)

					rpcHandlers.ApiEndpointStub = func(_ string, retVal *string) error {
						*retVal = ""
						return nil
					}
				})

				It("Failed with missing cf api setting", func() {
					args = []string{ts.Port(), "autoscaling-metrics", fakeAppName, metricName}
					session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()
					Expect(session).To(gbytes.Say(ui.NOCFAPIEndpoint))
					Expect(session.ExitCode()).To(Equal(1))
				})
			})

			Context("When cf api is changed ", func() {
				BeforeEach(func() {
					urlConfig := []byte(fmt.Sprintf(`{"URL":"%s"}`, "autoscaler.bosh-lite.com"))
					err = ioutil.WriteFile(api.ConfigFile(), urlConfig, 0600)
					Expect(err).NotTo(HaveOccurred())

				})

				Context("When the default endpoint doesn't work", func() {

					BeforeEach(func() {
						apiServer.RouteToHandler("GET", "/health",
							ghttp.RespondWith(http.StatusNotFound, ""),
						)
					})

					It("Failed with no api endpoint setting", func() {
						args = []string{ts.Port(), "autoscaling-metrics", fakeAppName, metricName}
						session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
						Expect(err).NotTo(HaveOccurred())
						session.Wait()
						Expect(session).To(gbytes.Say(ui.NoEndpoint))
						Expect(session.ExitCode()).To(Equal(1))
					})
				})
			})

			Context("when cf not login", func() {
				It("exits with 'You must be logged in' error ", func() {
					args = []string{ts.Port(), "autoscaling-metrics", fakeAppName, metricName}
					session, err := gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()
					Expect(session).To(gbytes.Say("You must be logged in"))
					Expect(session.ExitCode()).To(Equal(1))
				})
			})

			Context("when cf login", func() {
				BeforeEach(func() {
					rpcHandlers.IsLoggedInStub = func(args string, retVal *bool) error {
						*retVal = true
						return nil
					}
				})

				Context("when app not found", func() {
					BeforeEach(func() {
						rpcHandlers.GetAppStub = func(_ string, retVal *plugin_models.GetAppModel) error {
							return errors.New("App fakeApp not found")
						}
					})

					It("exits with 'App not found' error ", func() {
						args = []string{ts.Port(), "autoscaling-metrics", fakeAppName, metricName}
						session, err := gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
						Expect(err).NotTo(HaveOccurred())
						session.Wait()
						Expect(session).To(gbytes.Say("App fakeApp not found"))
						Expect(session.ExitCode()).To(Equal(1))
					})
				})

				Context("when the app is found", func() {
					BeforeEach(func() {
						rpcHandlers.GetAppStub = func(_ string, retVal *plugin_models.GetAppModel) error {
							*retVal = plugin_models.GetAppModel{
								Guid: fakeAppId,
							}
							return nil
						}
					})

					JustBeforeEach(func() {
						args = []string{ts.Port(), "autoscaling-api", apiEndpoint}
						session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
						Expect(err).NotTo(HaveOccurred())
						session.Wait()
					})

					Context("when access token is wrong", func() {
						BeforeEach(func() {
							rpcHandlers.AccessTokenStub = func(args string, retVal *string) error {
								*retVal = "incorrectAccessToken"
								return nil
							}

							apiServer.RouteToHandler("GET", aggregatedMetricsUrlPath,
								ghttp.CombineHandlers(
									ghttp.RespondWith(http.StatusUnauthorized, ""),
								),
							)
						})

						It("failed with 401 error", func() {
							args = []string{ts.Port(), "autoscaling-metrics", fakeAppName, metricName}
							session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
							Expect(err).NotTo(HaveOccurred())
							session.Wait()

							Expect(session).To(gbytes.Say("Failed to access AutoScaler API endpoint"))
							Expect(session.ExitCode()).To(Equal(1))
						})
					})

					Context("when access token is correct", func() {

						BeforeEach(func() {
							rpcHandlers.AccessTokenStub = func(args string, retVal *string) error {
								*retVal = fakeAccessToken
								return nil
							}
						})
						Context("when no aggregated metric record in desired duration", func() {
							BeforeEach(func() {
								apiServer.RouteToHandler("GET", aggregatedMetricsUrlPath,
									ghttp.CombineHandlers(
										ghttp.RespondWithJSONEncoded(http.StatusOK, &AggregatedMetricsResults{
											TotalResults: 0,
											TotalPages:   0,
											Page:         1,
											Metrics:      []*AppAggregatedMetric{},
										}),
										ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
									),
								)
							})

							It("Succeed and return no record", func() {
								args = []string{ts.Port(), "autoscaling-metrics", fakeAppName, metricName,
									"--start", now.Format(time.RFC3339),
									"--end", time.Unix(0, lowPrecisionNowInNano+int64(9*30*1E9)).Format(time.RFC3339)}

								session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
								Expect(err).NotTo(HaveOccurred())
								session.Wait()

								Expect(session).To(gbytes.Say("OK"))
								Expect(session).To(gbytes.Say(ui.AggregatedMetricsNotFound, metricName, fakeAppName))
								Expect(session.ExitCode()).To(Equal(0))

							})
						})

						Context("when metrics are available", func() {
							var metrics, reversedMetrics []*AppAggregatedMetric

							BeforeEach(func() {
								for i := 0; i < 30; i++ {
									metrics = append(metrics, &AppAggregatedMetric{
										AppId:     fakeAppId,
										Name:      "memoryused",
										Unit:      "MB",
										Value:     "100",
										Timestamp: now.UnixNano() + int64(i*30*1E9),
									})
								}

								for i := 0; i < 30; i++ {
									reversedMetrics = append(reversedMetrics, metrics[len(metrics)-1-i])
								}

							})
							Context("Query with default options ", func() {

								BeforeEach(func() {
									apiServer.AppendHandlers(
										ghttp.CombineHandlers(
											ghttp.RespondWithJSONEncoded(http.StatusOK, &AggregatedMetricsResults{
												TotalResults: 20,
												TotalPages:   2,
												Page:         1,
												Metrics:      metrics[0:10],
											}),
											ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
										),
									)
									apiServer.AppendHandlers(
										ghttp.CombineHandlers(
											ghttp.RespondWithJSONEncoded(http.StatusOK, &AggregatedMetricsResults{
												TotalResults: 20,
												TotalPages:   2,
												Page:         2,
												Metrics:      metrics[10:20],
											}),
											ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
										),
									)

								})

								It("Succeed to print first page of the metrics to stdout with asc order", func() {

									args = []string{ts.Port(), "autoscaling-metrics", fakeAppName, metricName}

									session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
									Expect(err).NotTo(HaveOccurred())
									session.Wait()

									Expect(session.Out).To(gbytes.Say(ui.ShowAggregatedMetricsHint, metricName, fakeAppName))
									metricsRaw := bytes.TrimPrefix(session.Out.Contents(), []byte(fmt.Sprintf(ui.ShowAggregatedMetricsHint+"\n", metricName, fakeAppName)))
									metricsTable := strings.Split(string(bytes.TrimRight(metricsRaw, "\n")), "\n")
									Expect(len(metricsTable)).To(Equal(12))
									for i, row := range metricsTable {
										colomns := strings.Split(row, "\t")
										if i == 0 {
											Expect(strings.Trim(colomns[0], " ")).To(Equal("Metrics Name"))
											Expect(strings.Trim(colomns[1], " ")).To(Equal("Value"))
											Expect(strings.Trim(colomns[2], " ")).To(Equal("Timestamp"))
										} else if i != len(metricsTable)-1 {
											Expect(strings.Trim(colomns[0], " ")).To(Equal("memoryused"))
											Expect(strings.Trim(colomns[1], " ")).To(Equal("100MB"))
											Expect(strings.Trim(colomns[2], " ")).To(Equal(time.Unix(0, now.UnixNano()+int64((i-1)*30*1E9)).Format(time.RFC3339)))
										}
									}
									Expect(session.ExitCode()).To(Equal(0))
								})

							})

							Context("Query multiple pages with asc order ", func() {

								Context("specifiy --start and --end both ", func() {
									BeforeEach(func() {
										//simulate the asc response from api server
										apiServer.AppendHandlers(
											ghttp.CombineHandlers(
												ghttp.RespondWithJSONEncoded(http.StatusOK, &AggregatedMetricsResults{
													TotalResults: 30,
													TotalPages:   3,
													Page:         1,
													Metrics:      metrics[0:10],
												}),
												ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
												ghttp.VerifyRequest("GET", aggregatedMetricsUrlPath,
													fmt.Sprintf("order=asc&page=1&start-time=%v&end-time=%v", lowPrecisionNowInNano, lowPrecisionNowInNano+int64(29*30*1E9)),
												),
											),
										)
										apiServer.AppendHandlers(
											ghttp.CombineHandlers(
												ghttp.RespondWithJSONEncoded(http.StatusOK, &AggregatedMetricsResults{
													TotalResults: 30,
													TotalPages:   3,
													Page:         2,
													Metrics:      metrics[10:20],
												}),
												ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
												ghttp.VerifyRequest("GET", aggregatedMetricsUrlPath,
													fmt.Sprintf("order=asc&page=2&start-time=%v&end-time=%v", lowPrecisionNowInNano, lowPrecisionNowInNano+int64(29*30*1E9)),
												),
											),
										)
										apiServer.AppendHandlers(
											ghttp.CombineHandlers(
												ghttp.RespondWithJSONEncoded(http.StatusOK, &AggregatedMetricsResults{
													TotalResults: 30,
													TotalPages:   3,
													Page:         3,
													Metrics:      metrics[20:30],
												}),
												ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
												ghttp.VerifyRequest("GET", aggregatedMetricsUrlPath,
													fmt.Sprintf("order=asc&page=3&start-time=%v&end-time=%v", lowPrecisionNowInNano, lowPrecisionNowInNano+int64(29*30*1E9)),
												),
											),
										)

									})

									It("Succeed to print all pages of the metrics to stdout with asc order", func() {

										args = []string{ts.Port(), "autoscaling-metrics", fakeAppName, metricName,
											"--start", now.Format(time.RFC3339),
											"--end", time.Unix(0, lowPrecisionNowInNano+int64(29*30*1E9)).Format(time.RFC3339)}

										session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
										Expect(err).NotTo(HaveOccurred())
										session.Wait()

										Expect(session.Out).To(gbytes.Say(ui.ShowAggregatedMetricsHint, metricName, fakeAppName))
										metricsRaw := bytes.TrimPrefix(session.Out.Contents(), []byte(fmt.Sprintf(ui.ShowAggregatedMetricsHint+"\n", metricName, fakeAppName)))
										metricsTable := strings.Split(string(bytes.TrimRight(metricsRaw, "\n")), "\n")
										Expect(len(metricsTable)).To(Equal(31))
										for i, row := range metricsTable {
											colomns := strings.Split(row, "\t")
											if i == 0 {
												Expect(strings.Trim(colomns[0], " ")).To(Equal("Metrics Name"))
												Expect(strings.Trim(colomns[1], " ")).To(Equal("Value"))
												Expect(strings.Trim(colomns[2], " ")).To(Equal("Timestamp"))
											} else {
												Expect(strings.Trim(colomns[0], " ")).To(Equal("memoryused"))
												Expect(strings.Trim(colomns[1], " ")).To(Equal("100MB"))
												Expect(strings.Trim(colomns[2], " ")).To(Equal(time.Unix(0, now.UnixNano()+int64((i-1)*30*1E9)).Format(time.RFC3339)))
											}
										}
										Expect(session.ExitCode()).To(Equal(0))
									})
								})

								Context("specifiy --end only ", func() {
									BeforeEach(func() {
										//simulate the asc response from api server
										apiServer.AppendHandlers(
											ghttp.CombineHandlers(
												ghttp.RespondWithJSONEncoded(http.StatusOK, &AggregatedMetricsResults{
													TotalResults: 30,
													TotalPages:   3,
													Page:         1,
													Metrics:      metrics[0:10],
												}),
												ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
												ghttp.VerifyRequest("GET", aggregatedMetricsUrlPath,
													fmt.Sprintf("order=asc&page=1&end-time=%v", lowPrecisionNowInNano+int64(29*30*1E9)),
												),
											),
										)
										apiServer.AppendHandlers(
											ghttp.CombineHandlers(
												ghttp.RespondWithJSONEncoded(http.StatusOK, &AggregatedMetricsResults{
													TotalResults: 30,
													TotalPages:   3,
													Page:         2,
													Metrics:      metrics[10:20],
												}),
												ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
												ghttp.VerifyRequest("GET", aggregatedMetricsUrlPath,
													fmt.Sprintf("order=asc&page=2&end-time=%v", lowPrecisionNowInNano+int64(29*30*1E9)),
												),
											),
										)
										apiServer.AppendHandlers(
											ghttp.CombineHandlers(
												ghttp.RespondWithJSONEncoded(http.StatusOK, &AggregatedMetricsResults{
													TotalResults: 30,
													TotalPages:   3,
													Page:         3,
													Metrics:      metrics[20:30],
												}),
												ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
												ghttp.VerifyRequest("GET", aggregatedMetricsUrlPath,
													fmt.Sprintf("order=asc&page=3&end-time=%v", lowPrecisionNowInNano+int64(29*30*1E9)),
												),
											),
										)

									})

									It("Succeed to print all pages of the metrics to stdout with asc order", func() {

										args = []string{ts.Port(), "autoscaling-metrics", fakeAppName, metricName,
											"--end", time.Unix(0, lowPrecisionNowInNano+int64(29*30*1E9)).Format(time.RFC3339)}

										session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
										Expect(err).NotTo(HaveOccurred())
										session.Wait()

										Expect(session.Out).To(gbytes.Say(ui.ShowAggregatedMetricsHint, metricName, fakeAppName))
										metricsRaw := bytes.TrimPrefix(session.Out.Contents(), []byte(fmt.Sprintf(ui.ShowAggregatedMetricsHint+"\n", metricName, fakeAppName)))
										metricsTable := strings.Split(string(bytes.TrimRight(metricsRaw, "\n")), "\n")
										Expect(len(metricsTable)).To(Equal(31))
										for i, row := range metricsTable {
											colomns := strings.Split(row, "\t")
											if i == 0 {
												Expect(strings.Trim(colomns[0], " ")).To(Equal("Metrics Name"))
												Expect(strings.Trim(colomns[1], " ")).To(Equal("Value"))
												Expect(strings.Trim(colomns[2], " ")).To(Equal("Timestamp"))
											} else {
												Expect(strings.Trim(colomns[0], " ")).To(Equal("memoryused"))
												Expect(strings.Trim(colomns[1], " ")).To(Equal("100MB"))
												Expect(strings.Trim(colomns[2], " ")).To(Equal(time.Unix(0, now.UnixNano()+int64((i-1)*30*1E9)).Format(time.RFC3339)))
											}
										}
										Expect(session.ExitCode()).To(Equal(0))
									})
								})
							})

							Context("Query multiple pages with desc order ", func() {

								BeforeEach(func() {
									//simulate the desc response from api server
									apiServer.AppendHandlers(
										ghttp.CombineHandlers(
											ghttp.RespondWithJSONEncoded(http.StatusOK, &AggregatedMetricsResults{
												TotalResults: 30,
												TotalPages:   3,
												Page:         1,
												Metrics:      reversedMetrics[0:10],
											}),
											ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
											ghttp.VerifyRequest("GET", aggregatedMetricsUrlPath,
												fmt.Sprintf("order=desc&page=1&start-time=%v&end-time=%v", lowPrecisionNowInNano, lowPrecisionNowInNano+int64(29*30*1E9)),
											),
										),
									)
									apiServer.AppendHandlers(
										ghttp.CombineHandlers(
											ghttp.RespondWithJSONEncoded(http.StatusOK, &AggregatedMetricsResults{
												TotalResults: 30,
												TotalPages:   3,
												Page:         2,
												Metrics:      reversedMetrics[10:20],
											}),
											ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
											ghttp.VerifyRequest("GET", aggregatedMetricsUrlPath,
												fmt.Sprintf("order=desc&page=2&start-time=%v&end-time=%v", lowPrecisionNowInNano, lowPrecisionNowInNano+int64(29*30*1E9)),
											),
										),
									)
									apiServer.AppendHandlers(
										ghttp.CombineHandlers(
											ghttp.RespondWithJSONEncoded(http.StatusOK, &AggregatedMetricsResults{
												TotalResults: 30,
												TotalPages:   3,
												Page:         3,
												Metrics:      reversedMetrics[20:30],
											}),
											ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
											ghttp.VerifyRequest("GET", aggregatedMetricsUrlPath,
												fmt.Sprintf("order=desc&page=3&start-time=%v&end-time=%v", lowPrecisionNowInNano, lowPrecisionNowInNano+int64(29*30*1E9)),
											),
										),
									)

								})

								It("Succeed to print all pages of the metrics to stdout with desc order", func() {

									args = []string{ts.Port(), "autoscaling-metrics", fakeAppName, metricName,
										"--start", now.Format(time.RFC3339),
										"--end", time.Unix(0, lowPrecisionNowInNano+int64(29*30*1E9)).Format(time.RFC3339),
										"--desc",
									}

									session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
									Expect(err).NotTo(HaveOccurred())
									session.Wait()

									Expect(session.Out).To(gbytes.Say(ui.ShowAggregatedMetricsHint, metricName, fakeAppName))
									metricsRaw := bytes.TrimPrefix(session.Out.Contents(), []byte(fmt.Sprintf(ui.ShowAggregatedMetricsHint+"\n", metricName, fakeAppName)))
									metricsTable := strings.Split(string(bytes.TrimRight(metricsRaw, "\n")), "\n")
									Expect(len(metricsTable)).To(Equal(31))
									for i, row := range metricsTable {
										colomns := strings.Split(row, "\t")
										if i == 0 {
											Expect(strings.Trim(colomns[0], " ")).To(Equal("Metrics Name"))
											Expect(strings.Trim(colomns[1], " ")).To(Equal("Value"))
											Expect(strings.Trim(colomns[2], " ")).To(Equal("Timestamp"))
										} else {
											//use "29-(i-1)" to simulate the expected output in desc order
											Expect(strings.Trim(colomns[0], " ")).To(Equal("memoryused"))
											Expect(strings.Trim(colomns[1], " ")).To(Equal("100MB"))
											Expect(strings.Trim(colomns[2], " ")).To(Equal(time.Unix(0, now.UnixNano()+int64((29-(i-1))*30*1E9)).Format(time.RFC3339)))
										}
									}
									Expect(session.ExitCode()).To(Equal(0))
								})

							})

							Context(" Print the output to a file", func() {

								BeforeEach(func() {
									apiServer.RouteToHandler("GET", aggregatedMetricsUrlPath,
										ghttp.CombineHandlers(
											ghttp.RespondWithJSONEncoded(http.StatusOK, &AggregatedMetricsResults{
												TotalResults: 10,
												TotalPages:   1,
												Page:         1,
												Metrics:      metrics[0:10],
											}),
											ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
										),
									)

								})

								It("Succeed to print the metrics to stdout with asc order", func() {

									args = []string{ts.Port(), "autoscaling-metrics", fakeAppName, metricName, "--output", outputFile}

									session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
									Expect(err).NotTo(HaveOccurred())
									session.Wait()

									Expect(session.Out).To(gbytes.Say(ui.SaveAggregatedMetricHint, fakeAppName, outputFile))
									Expect(session.Out).To(gbytes.Say("OK"))

									Expect(outputFile).To(BeARegularFile())
									contents, err := ioutil.ReadFile(outputFile)
									Expect(err).NotTo(HaveOccurred())

									metricsTable := strings.Split(string(bytes.TrimRight(contents, "\n")), "\n")
									for i, row := range metricsTable {
										colomns := strings.Split(row, "\t")
										if i == 0 {
											Expect(strings.Trim(colomns[0], " ")).To(Equal("Metrics Name"))
											Expect(strings.Trim(colomns[1], " ")).To(Equal("Value"))
											Expect(strings.Trim(colomns[2], " ")).To(Equal("Timestamp"))
										} else {
											Expect(strings.Trim(colomns[0], " ")).To(Equal("memoryused"))
											Expect(strings.Trim(colomns[1], " ")).To(Equal("100MB"))
											Expect(strings.Trim(colomns[2], " ")).To(Equal(time.Unix(0, now.UnixNano()+int64((i-1)*30*1E9)).Format(time.RFC3339)))
										}
									}
									Expect(session.ExitCode()).To(Equal(0))
								})

							})

						})

					})

				})

			})
		})
	})

	Describe("Commands autoscaling-history, ash", func() {

		var (
			urlpath               = "/v1/apps/" + fakeAppId + "/scaling_histories"
			now                   = time.Now()
			lowPrecisionNowInNano = (now.UnixNano() / 1E9) * 1E9
		)

		Context("autoscaling-history", func() {

			Context("when the args or options are not properly provided", func() {

				It("Require APP_NAME as argument", func() {
					args = []string{ts.Port(), "autoscaling-history"}
					session, err := gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()

					Expect(session).To(gbytes.Say("the required argument `APP_NAME` was not provided"))
					Expect(session.ExitCode()).To(Equal(1))
				})

				It("Failed when start/end time is defined in unsupported time format", func() {
					invalidTime := now.Format(time.UnixDate)
					args = []string{ts.Port(), "autoscaling-history", fakeAppName, "--start", invalidTime}
					session, err := gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()

					Expect(session).To(gbytes.Say("Unrecognized date time format"))
					Expect(session.ExitCode()).To(Equal(1))

					args = []string{ts.Port(), "autoscaling-history", fakeAppName, "--end", invalidTime}
					session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()

					Expect(session).To(gbytes.Say("Unrecognized date time format"))
					Expect(session.ExitCode()).To(Equal(1))
				})

				It("Failed when start/end time is prior to 1970-01-01T00:00:00Z", func() {
					args = []string{ts.Port(), "autoscaling-history", fakeAppName,
						"--start", "1969-12-31-T00:00:00Z",
						"--end", "1969-12-31-T23:59:59Z",
					}
					session, err := gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()

					Expect(session).To(gbytes.Say("Unrecognized date time format"))
					Expect(session.ExitCode()).To(Equal(1))
				})

				It("Failed when start time is greater than end time", func() {
					args = []string{ts.Port(), "autoscaling-history", fakeAppName,
						"--start", now.Format(time.RFC3339),
						"--end", time.Unix(0, now.UnixNano()-int64(30*1E9)).Format(time.RFC3339),
					}
					session, err := gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()

					expects := strings.Split(ui.InvalidTimeRange, "%s")
					for _, expect := range expects {
						Expect(session).To(gbytes.Say(expect))
					}
					Expect(session.ExitCode()).To(Equal(1))
				})

				It("Failed when --desc is wrong spelled", func() {
					args = []string{ts.Port(), "autoscaling-history", fakeAppName, "--dddesc"}
					session, err := gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()

					Expect(session).To(gbytes.Say("unknown flag"))
					Expect(session.ExitCode()).To(Equal(1))
				})

				It("Failed when output file path is invalid", func() {
					args = []string{ts.Port(), "autoscaling-history", fakeAppName, "--output", "invalidDir/invalidFile"}
					session, err := gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()

					Expect(session).To(gbytes.Say("open invalidDir/invalidFile: no such file or directory"))
					Expect(session.ExitCode()).To(Equal(1))
				})

			})

			Context("When cf api is not set ", func() {
				BeforeEach(func() {

					rpcHandlers.ApiEndpointStub = func(_ string, retVal *string) error {
						*retVal = ""
						return nil
					}

					args = []string{ts.Port(), "autoscaling-api", apiEndpoint}
					session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)

					rpcHandlers.ApiEndpointStub = func(_ string, retVal *string) error {
						*retVal = ""
						return nil
					}
				})

				It("Failed with missing cf api setting", func() {
					args = []string{ts.Port(), "autoscaling-history", fakeAppName}
					session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()
					Expect(session).To(gbytes.Say(ui.NOCFAPIEndpoint))
					Expect(session.ExitCode()).To(Equal(1))
				})
			})

			Context("When cf api is changed ", func() {
				BeforeEach(func() {
					urlConfig := []byte(fmt.Sprintf(`{"URL":"%s"}`, "autoscaler.bosh-lite.com"))
					err = ioutil.WriteFile(api.ConfigFile(), urlConfig, 0600)
					Expect(err).NotTo(HaveOccurred())

				})

				Context("When the default endpoint doesn't work", func() {

					BeforeEach(func() {
						apiServer.RouteToHandler("GET", "/health",
							ghttp.RespondWith(http.StatusNotFound, ""),
						)
					})

					It("Failed with no api endpoint setting", func() {
						args = []string{ts.Port(), "autoscaling-history", fakeAppName}
						session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
						Expect(err).NotTo(HaveOccurred())
						session.Wait()
						Expect(session).To(gbytes.Say(ui.NoEndpoint))
						Expect(session.ExitCode()).To(Equal(1))
					})
				})
			})

			Context("when cf not login", func() {
				It("exits with 'You must be logged in' error ", func() {
					args = []string{ts.Port(), "autoscaling-history", fakeAppName}
					session, err := gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()
					Expect(session).To(gbytes.Say("You must be logged in"))
					Expect(session.ExitCode()).To(Equal(1))
				})
			})

			Context("when cf login", func() {
				BeforeEach(func() {
					rpcHandlers.IsLoggedInStub = func(args string, retVal *bool) error {
						*retVal = true
						return nil
					}
				})

				Context("when app not found", func() {
					BeforeEach(func() {
						rpcHandlers.GetAppStub = func(_ string, retVal *plugin_models.GetAppModel) error {
							return errors.New("App fakeApp not found")
						}
					})

					It("exits with 'App not found' error ", func() {
						args = []string{ts.Port(), "autoscaling-history", fakeAppName}
						session, err := gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
						Expect(err).NotTo(HaveOccurred())
						session.Wait()
						Expect(session).To(gbytes.Say("App fakeApp not found"))
						Expect(session.ExitCode()).To(Equal(1))
					})
				})

				Context("when the app is found", func() {
					BeforeEach(func() {
						rpcHandlers.GetAppStub = func(_ string, retVal *plugin_models.GetAppModel) error {
							*retVal = plugin_models.GetAppModel{
								Guid: fakeAppId,
							}
							return nil
						}
					})

					JustBeforeEach(func() {
						args = []string{ts.Port(), "autoscaling-api", apiEndpoint}
						session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
						Expect(err).NotTo(HaveOccurred())
						session.Wait()
					})

					Context("when access token is wrong", func() {
						BeforeEach(func() {
							rpcHandlers.AccessTokenStub = func(args string, retVal *string) error {
								*retVal = "incorrectAccessToken"
								return nil
							}

							apiServer.RouteToHandler("GET", urlpath,
								ghttp.CombineHandlers(
									ghttp.RespondWith(http.StatusUnauthorized, ""),
								),
							)
						})

						It("failed with 401 error", func() {
							args = []string{ts.Port(), "autoscaling-history", fakeAppName}
							session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
							Expect(err).NotTo(HaveOccurred())
							session.Wait()

							Expect(session).To(gbytes.Say("Failed to access AutoScaler API endpoint"))
							Expect(session.ExitCode()).To(Equal(1))
						})
					})

					Context("when access token is correct", func() {

						BeforeEach(func() {
							rpcHandlers.AccessTokenStub = func(args string, retVal *string) error {
								*retVal = fakeAccessToken
								return nil
							}
						})

						Context("when no history record in desired duration", func() {
							BeforeEach(func() {

								apiServer.RouteToHandler("GET", urlpath,
									ghttp.CombineHandlers(
										ghttp.RespondWithJSONEncoded(http.StatusOK, &HistoryResults{
											TotalResults: 0,
											TotalPages:   0,
											Page:         1,
											Histories:    []*AppScalingHistory{},
										}),
										ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
									),
								)
							})

							It("Succeed but no record returned", func() {
								args = []string{ts.Port(), "autoscaling-history", fakeAppName,
									"--start", now.Format(time.RFC3339),
									"--end", time.Unix(0, lowPrecisionNowInNano+int64(9*120*1E9)).Format(time.RFC3339)}

								session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
								Expect(err).NotTo(HaveOccurred())
								session.Wait()

								Expect(session).To(gbytes.Say("OK"))
								Expect(session).To(gbytes.Say(ui.HistoryNotFound, fakeAppName))
								Expect(session.ExitCode()).To(Equal(0))

							})
						})

						Context("when history record are available", func() {
							var histories, reversedHistories []*AppScalingHistory

							BeforeEach(func() {
								for i := 0; i < 10; i++ {
									histories = append(histories, &AppScalingHistory{
										AppId:        fakeAppId,
										Timestamp:    now.UnixNano() + int64(i*120*1E9),
										ScalingType:  0, //dynamic
										Status:       0, //succeed
										OldInstances: i + 1,
										NewInstances: i + 2,
										Reason:       "fakeReason",
										Message:      "",
										Error:        "fakeError",
									})
								}

								for i := 10; i < 20; i++ {
									histories = append(histories, &AppScalingHistory{
										AppId:        fakeAppId,
										Timestamp:    now.UnixNano() + int64(i*120*1E9),
										ScalingType:  1, //scheduled
										Status:       0, //succeed
										OldInstances: i + 1,
										NewInstances: i + 2,
										Reason:       "fakeReason",
										Message:      "",
										Error:        "fakeError",
									})
								}

								for i := 20; i < 30; i++ {
									histories = append(histories, &AppScalingHistory{
										AppId:        fakeAppId,
										Timestamp:    now.UnixNano() + int64(i*120*1E9),
										ScalingType:  1, //scheduled
										Status:       1, //failed
										OldInstances: i + 1,
										NewInstances: i + 2,
										Reason:       "fakeReason",
										Message:      "",
										Error:        "fakeError",
									})
								}

								for i := 0; i < 30; i++ {
									reversedHistories = append(reversedHistories, histories[len(histories)-1-i])
								}

							})
							Context("Query with default options ", func() {

								BeforeEach(func() {
									apiServer.AppendHandlers(
										ghttp.CombineHandlers(
											ghttp.RespondWithJSONEncoded(http.StatusOK, &HistoryResults{
												TotalResults: 20,
												TotalPages:   2,
												Page:         1,
												Histories:    histories[0:10],
											}),
											ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
										),
									)
									apiServer.AppendHandlers(
										ghttp.CombineHandlers(
											ghttp.RespondWithJSONEncoded(http.StatusOK, &HistoryResults{
												TotalResults: 20,
												TotalPages:   2,
												Page:         2,
												Histories:    histories[10:20],
											}),
											ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
										),
									)
								})

								It("Succeed to print first page of the histories to stdout with asc order", func() {

									args = []string{ts.Port(), "autoscaling-history", fakeAppName}

									session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
									Expect(err).NotTo(HaveOccurred())
									session.Wait()

									Expect(session.Out).To(gbytes.Say(ui.ShowHistoryHint, fakeAppName))
									historyRaw := bytes.TrimPrefix(session.Out.Contents(), []byte(fmt.Sprintf(ui.ShowHistoryHint+"\n", fakeAppName)))
									historyTable := strings.Split(string(bytes.TrimRight(historyRaw, "\n")), "\n")
									Expect(len(historyTable)).To(Equal(12))
									for i, row := range historyTable {
										colomns := strings.Split(row, "\t")
										if i == 0 {
											Expect(strings.Trim(colomns[0], " ")).To(Equal("Scaling Type"))
											Expect(strings.Trim(colomns[1], " ")).To(Equal("Status"))
											Expect(strings.Trim(colomns[2], " ")).To(Equal("Instance Changes"))
											Expect(strings.Trim(colomns[3], " ")).To(Equal("Time"))
											Expect(strings.Trim(colomns[4], " ")).To(Equal("Action"))
											Expect(strings.Trim(colomns[5], " ")).To(Equal("Error"))

										} else if i != len(historyTable)-1 {
											Expect(strings.Trim(colomns[0], " ")).To(Equal("dynamic"))
											Expect(strings.Trim(colomns[1], " ")).To(Equal("succeeded"))
											Expect(strings.Trim(colomns[2], " ")).To(Equal(strconv.Itoa(i-1+1) + "->" + strconv.Itoa(i-1+2)))
											Expect(strings.Trim(colomns[3], " ")).To(Equal(time.Unix(0, now.UnixNano()+int64((i-1)*120*1E9)).Format(time.RFC3339)))
											Expect(strings.Trim(colomns[4], " ")).To(Equal("fakeReason"))
											Expect(strings.Trim(colomns[5], " ")).To(Equal("fakeError"))
										}
									}
									Expect(session.ExitCode()).To(Equal(0))
								})

							})

							Context("Query multiple pages with asc order ", func() {

								Context("specifiy --start and --end both ", func() {
									BeforeEach(func() {
										//simulate the asc response from api server
										apiServer.AppendHandlers(
											ghttp.CombineHandlers(
												ghttp.RespondWithJSONEncoded(http.StatusOK, &HistoryResults{
													TotalResults: 30,
													TotalPages:   3,
													Page:         1,
													Histories:    histories[0:10],
												}),
												ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
												ghttp.VerifyRequest("GET", urlpath,
													fmt.Sprintf("order=asc&page=1&start-time=%v&end-time=%v", lowPrecisionNowInNano, lowPrecisionNowInNano+int64(29*120*1E9)),
												),
											),
										)
										apiServer.AppendHandlers(
											ghttp.CombineHandlers(
												ghttp.RespondWithJSONEncoded(http.StatusOK, &HistoryResults{
													TotalResults: 30,
													TotalPages:   3,
													Page:         2,
													Histories:    histories[10:20],
												}),
												ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
												ghttp.VerifyRequest("GET", urlpath,
													fmt.Sprintf("order=asc&page=2&start-time=%v&end-time=%v", lowPrecisionNowInNano, lowPrecisionNowInNano+int64(29*120*1E9)),
												),
											),
										)
										apiServer.AppendHandlers(
											ghttp.CombineHandlers(
												ghttp.RespondWithJSONEncoded(http.StatusOK, &HistoryResults{
													TotalResults: 30,
													TotalPages:   3,
													Page:         3,
													Histories:    histories[20:30],
												}),
												ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
												ghttp.VerifyRequest("GET", urlpath,
													fmt.Sprintf("order=asc&page=3&start-time=%v&end-time=%v", lowPrecisionNowInNano, lowPrecisionNowInNano+int64(29*120*1E9)),
												),
											),
										)

									})

									It("Succeed to print all pages of the history to stdout", func() {

										args = []string{ts.Port(), "autoscaling-history", fakeAppName,
											"--start", now.Format(time.RFC3339),
											"--end", time.Unix(0, lowPrecisionNowInNano+int64(29*120*1E9)).Format(time.RFC3339)}

										session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
										Expect(err).NotTo(HaveOccurred())
										session.Wait()

										Expect(session.Out).To(gbytes.Say(ui.ShowHistoryHint, fakeAppName))
										historyRaw := bytes.TrimPrefix(session.Out.Contents(), []byte(fmt.Sprintf(ui.ShowHistoryHint+"\n", fakeAppName)))
										historyTable := strings.Split(string(bytes.TrimRight(historyRaw, "\n")), "\n")
										Expect(len(historyTable)).To(Equal(31))
										for i, row := range historyTable {
											colomns := strings.Split(row, "\t")
											if i == 0 {
												Expect(strings.Trim(colomns[0], " ")).To(Equal("Scaling Type"))
												Expect(strings.Trim(colomns[1], " ")).To(Equal("Status"))
												Expect(strings.Trim(colomns[2], " ")).To(Equal("Instance Changes"))
												Expect(strings.Trim(colomns[3], " ")).To(Equal("Time"))
												Expect(strings.Trim(colomns[4], " ")).To(Equal("Action"))
												Expect(strings.Trim(colomns[5], " ")).To(Equal("Error"))
												//header line
											} else {
												//use (i-1) to skip header
												Expect(strings.Trim(colomns[3], " ")).To(Equal(time.Unix(0, now.UnixNano()+int64((i-1)*120*1E9)).Format(time.RFC3339)))
												Expect(strings.Trim(colomns[4], " ")).To(Equal("fakeReason"))
												Expect(strings.Trim(colomns[5], " ")).To(Equal("fakeError"))

												if i < 11 {
													Expect(strings.Trim(colomns[0], " ")).To(Equal("dynamic"))
													Expect(strings.Trim(colomns[1], " ")).To(Equal("succeeded"))
													Expect(strings.Trim(colomns[2], " ")).To(Equal(strconv.Itoa(i-1+1) + "->" + strconv.Itoa(i-1+2)))
												} else if i < 21 {
													Expect(strings.Trim(colomns[0], " ")).To(Equal("scheduled"))
													Expect(strings.Trim(colomns[1], " ")).To(Equal("succeeded"))
													Expect(strings.Trim(colomns[2], " ")).To(Equal(strconv.Itoa(i-1+1) + "->" + strconv.Itoa(i-1+2)))
												} else {
													Expect(strings.Trim(colomns[0], " ")).To(Equal("scheduled"))
													Expect(strings.Trim(colomns[1], " ")).To(Equal("failed"))
													Expect(strings.Trim(colomns[2], " ")).To(Equal(""))
												}
											}
										}
										Expect(session.ExitCode()).To(Equal(0))
									})
								})

								Context("specify --end only ", func() {
									BeforeEach(func() {
										//simulate the asc response from api server
										apiServer.AppendHandlers(
											ghttp.CombineHandlers(
												ghttp.RespondWithJSONEncoded(http.StatusOK, &HistoryResults{
													TotalResults: 30,
													TotalPages:   3,
													Page:         1,
													Histories:    histories[0:10],
												}),
												ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
												ghttp.VerifyRequest("GET", urlpath,
													fmt.Sprintf("order=asc&page=1&end-time=%v", lowPrecisionNowInNano+int64(29*120*1E9)),
												),
											),
										)
										apiServer.AppendHandlers(
											ghttp.CombineHandlers(
												ghttp.RespondWithJSONEncoded(http.StatusOK, &HistoryResults{
													TotalResults: 30,
													TotalPages:   3,
													Page:         2,
													Histories:    histories[10:20],
												}),
												ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
												ghttp.VerifyRequest("GET", urlpath,
													fmt.Sprintf("order=asc&page=2&end-time=%v", lowPrecisionNowInNano+int64(29*120*1E9)),
												),
											),
										)
										apiServer.AppendHandlers(
											ghttp.CombineHandlers(
												ghttp.RespondWithJSONEncoded(http.StatusOK, &HistoryResults{
													TotalResults: 30,
													TotalPages:   3,
													Page:         3,
													Histories:    histories[20:30],
												}),
												ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
												ghttp.VerifyRequest("GET", urlpath,
													fmt.Sprintf("order=asc&page=3&end-time=%v", lowPrecisionNowInNano+int64(29*120*1E9)),
												),
											),
										)

									})

									It("Succeed to print all pages of the history to stdout", func() {

										args = []string{ts.Port(), "autoscaling-history", fakeAppName,
											"--end", time.Unix(0, lowPrecisionNowInNano+int64(29*120*1E9)).Format(time.RFC3339)}

										session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
										Expect(err).NotTo(HaveOccurred())
										session.Wait()

										Expect(session.Out).To(gbytes.Say(ui.ShowHistoryHint, fakeAppName))
										historyRaw := bytes.TrimPrefix(session.Out.Contents(), []byte(fmt.Sprintf(ui.ShowHistoryHint+"\n", fakeAppName)))
										historyTable := strings.Split(string(bytes.TrimRight(historyRaw, "\n")), "\n")
										Expect(len(historyTable)).To(Equal(31))
										for i, row := range historyTable {
											colomns := strings.Split(row, "\t")
											if i == 0 {
												Expect(strings.Trim(colomns[0], " ")).To(Equal("Scaling Type"))
												Expect(strings.Trim(colomns[1], " ")).To(Equal("Status"))
												Expect(strings.Trim(colomns[2], " ")).To(Equal("Instance Changes"))
												Expect(strings.Trim(colomns[3], " ")).To(Equal("Time"))
												Expect(strings.Trim(colomns[4], " ")).To(Equal("Action"))
												Expect(strings.Trim(colomns[5], " ")).To(Equal("Error"))
												//header line
											} else {
												//use (i-1) to skip header
												Expect(strings.Trim(colomns[3], " ")).To(Equal(time.Unix(0, now.UnixNano()+int64((i-1)*120*1E9)).Format(time.RFC3339)))
												Expect(strings.Trim(colomns[4], " ")).To(Equal("fakeReason"))
												Expect(strings.Trim(colomns[5], " ")).To(Equal("fakeError"))

												if i < 11 {
													Expect(strings.Trim(colomns[0], " ")).To(Equal("dynamic"))
													Expect(strings.Trim(colomns[1], " ")).To(Equal("succeeded"))
													Expect(strings.Trim(colomns[2], " ")).To(Equal(strconv.Itoa(i-1+1) + "->" + strconv.Itoa(i-1+2)))
												} else if i < 21 {
													Expect(strings.Trim(colomns[0], " ")).To(Equal("scheduled"))
													Expect(strings.Trim(colomns[1], " ")).To(Equal("succeeded"))
													Expect(strings.Trim(colomns[2], " ")).To(Equal(strconv.Itoa(i-1+1) + "->" + strconv.Itoa(i-1+2)))
												} else {
													Expect(strings.Trim(colomns[0], " ")).To(Equal("scheduled"))
													Expect(strings.Trim(colomns[1], " ")).To(Equal("failed"))
													Expect(strings.Trim(colomns[2], " ")).To(Equal(""))
												}
											}
										}
										Expect(session.ExitCode()).To(Equal(0))
									})
								})

							})

							Context("Query multiple pages with desc order ", func() {

								BeforeEach(func() {
									//simulate the desc response from api server
									apiServer.AppendHandlers(
										ghttp.CombineHandlers(
											ghttp.RespondWithJSONEncoded(http.StatusOK, &HistoryResults{
												TotalResults: 30,
												TotalPages:   3,
												Page:         1,
												Histories:    reversedHistories[0:10],
											}),
											ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
											ghttp.VerifyRequest("GET", urlpath,
												fmt.Sprintf("order=desc&page=1&start-time=%v&end-time=%v", lowPrecisionNowInNano, lowPrecisionNowInNano+int64(29*120*1E9)),
											),
										),
									)
									apiServer.AppendHandlers(
										ghttp.CombineHandlers(
											ghttp.RespondWithJSONEncoded(http.StatusOK, &HistoryResults{
												TotalResults: 30,
												TotalPages:   3,
												Page:         2,
												Histories:    reversedHistories[10:20],
											}),
											ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
											ghttp.VerifyRequest("GET", urlpath,
												fmt.Sprintf("order=desc&page=2&start-time=%v&end-time=%v", lowPrecisionNowInNano, lowPrecisionNowInNano+int64(29*120*1E9)),
											),
										),
									)
									apiServer.AppendHandlers(
										ghttp.CombineHandlers(
											ghttp.RespondWithJSONEncoded(http.StatusOK, &HistoryResults{
												TotalResults: 30,
												TotalPages:   3,
												Page:         3,
												Histories:    reversedHistories[20:30],
											}),
											ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
											ghttp.VerifyRequest("GET", urlpath,
												fmt.Sprintf("order=desc&page=3&start-time=%v&end-time=%v", lowPrecisionNowInNano, lowPrecisionNowInNano+int64(29*120*1E9)),
											),
										),
									)

								})

								It("Succeed to print all pages of the history to stdout with desc order", func() {

									args = []string{ts.Port(), "autoscaling-history", fakeAppName,
										"--start", now.Format(time.RFC3339),
										"--end", time.Unix(0, lowPrecisionNowInNano+int64(29*120*1E9)).Format(time.RFC3339),
										"--desc",
									}

									session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
									Expect(err).NotTo(HaveOccurred())
									session.Wait()

									Expect(session.Out).To(gbytes.Say(ui.ShowHistoryHint, fakeAppName))
									historyRaw := bytes.TrimPrefix(session.Out.Contents(), []byte(fmt.Sprintf(ui.ShowHistoryHint+"\n", fakeAppName)))
									historyTable := strings.Split(string(bytes.TrimRight(historyRaw, "\n")), "\n")
									Expect(len(historyTable)).To(Equal(31))
									for i, row := range historyTable {
										colomns := strings.Split(row, "\t")
										if i == 0 {
											Expect(strings.Trim(colomns[0], " ")).To(Equal("Scaling Type"))
											Expect(strings.Trim(colomns[1], " ")).To(Equal("Status"))
											Expect(strings.Trim(colomns[2], " ")).To(Equal("Instance Changes"))
											Expect(strings.Trim(colomns[3], " ")).To(Equal("Time"))
											Expect(strings.Trim(colomns[4], " ")).To(Equal("Action"))
											Expect(strings.Trim(colomns[5], " ")).To(Equal("Error"))
										} else {
											//use "29-(i-1)" to simulate the expected output in desc order
											Expect(strings.Trim(colomns[3], " ")).To(Equal(time.Unix(0, now.UnixNano()+int64((29-(i-1))*120*1E9)).Format(time.RFC3339)))
											Expect(strings.Trim(colomns[4], " ")).To(Equal("fakeReason"))
											Expect(strings.Trim(colomns[5], " ")).To(Equal("fakeError"))
											if i < 11 {
												Expect(strings.Trim(colomns[0], " ")).To(Equal("scheduled"))
												Expect(strings.Trim(colomns[1], " ")).To(Equal("failed"))
												Expect(strings.Trim(colomns[2], " ")).To(Equal(""))
											} else if i < 21 {
												Expect(strings.Trim(colomns[0], " ")).To(Equal("scheduled"))
												Expect(strings.Trim(colomns[1], " ")).To(Equal("succeeded"))
												Expect(strings.Trim(colomns[2], " ")).To(Equal(strconv.Itoa(29-(i-1)+1) + "->" + strconv.Itoa(29-(i-1)+2)))
											} else {
												Expect(strings.Trim(colomns[0], " ")).To(Equal("dynamic"))
												Expect(strings.Trim(colomns[1], " ")).To(Equal("succeeded"))
												Expect(strings.Trim(colomns[2], " ")).To(Equal(strconv.Itoa(29-(i-1)+1) + "->" + strconv.Itoa(29-(i-1)+2)))
											}
										}
									}
									Expect(session.ExitCode()).To(Equal(0))
								})

							})

							Context(" Print the output to a file", func() {

								BeforeEach(func() {
									apiServer.RouteToHandler("GET", urlpath,
										ghttp.CombineHandlers(
											ghttp.RespondWithJSONEncoded(http.StatusOK, &HistoryResults{
												TotalResults: 10,
												TotalPages:   1,
												Page:         1,
												Histories:    histories[0:10],
											}),
											ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
										),
									)

								})

								It("Succeed to print the history to a file with asc order", func() {

									args = []string{ts.Port(), "autoscaling-history", fakeAppName, "--output", outputFile}

									session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
									Expect(err).NotTo(HaveOccurred())
									session.Wait()

									Expect(session.Out).To(gbytes.Say(ui.SaveHistoryHint, fakeAppName, outputFile))
									Expect(session.Out).To(gbytes.Say("OK"))

									Expect(outputFile).To(BeARegularFile())
									contents, err := ioutil.ReadFile(outputFile)
									Expect(err).NotTo(HaveOccurred())

									historyTable := strings.Split(string(bytes.TrimRight(contents, "\n")), "\n")
									for i, row := range historyTable {
										colomns := strings.Split(row, "\t")
										if i == 0 {
											Expect(strings.Trim(colomns[0], " ")).To(Equal("Scaling Type"))
											Expect(strings.Trim(colomns[1], " ")).To(Equal("Status"))
											Expect(strings.Trim(colomns[2], " ")).To(Equal("Instance Changes"))
											Expect(strings.Trim(colomns[3], " ")).To(Equal("Time"))
											Expect(strings.Trim(colomns[4], " ")).To(Equal("Action"))
											Expect(strings.Trim(colomns[5], " ")).To(Equal("Error"))
										} else {
											Expect(strings.Trim(colomns[0], " ")).To(Equal("dynamic"))
											Expect(strings.Trim(colomns[1], " ")).To(Equal("succeeded"))
											Expect(strings.Trim(colomns[2], " ")).To(Equal(strconv.Itoa(i-1+1) + "->" + strconv.Itoa(i-1+2)))
											Expect(strings.Trim(colomns[3], " ")).To(Equal(time.Unix(0, now.UnixNano()+int64((i-1)*120*1E9)).Format(time.RFC3339)))
											Expect(strings.Trim(colomns[4], " ")).To(Equal("fakeReason"))
											Expect(strings.Trim(colomns[5], " ")).To(Equal("fakeError"))
										}
									}
									Expect(session.ExitCode()).To(Equal(0))
								})

							})

						})

					})

				})

			})
		})
	})

})
