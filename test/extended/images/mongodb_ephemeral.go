package images

import (
	"fmt"

	g "github.com/onsi/ginkgo"
	o "github.com/onsi/gomega"

	exutil "github.com/openshift/origin/test/extended/util"
	"github.com/openshift/origin/test/extended/util/db"
	"time"
)

var _ = g.Describe("images: mongodb: ephemeral template", func() {
	defer g.GinkgoRecover()

	templatePath := exutil.FixturePath("..", "..", "examples", "db-templates", "mongodb-ephemeral-template.json")
	oc := exutil.NewCLI("mongodb-create", exutil.KubeConfigPath()).Verbose()

	g.Describe("creating from a template", func() {
		g.It(fmt.Sprintf("should process and create the %q template", templatePath), func() {

			g.By("creating a new app")
			o.Expect(oc.Run("new-app").Args("-f", templatePath).Execute()).Should(o.Succeed())

			g.By("expecting the mongodb service get endpoints")
			o.Expect(oc.KubeFramework().WaitForAnEndpoint("mongodb")).Should(o.Succeed())

			g.By("expecting the mongodb pod is running")
			podNames, err := exutil.WaitForPods(
				oc.KubeREST().Pods(oc.Namespace()),
				exutil.ParseLabelsOrDie("name=mongodb"),
				exutil.CheckPodIsRunningFn,
				1,
				1*time.Minute,
			)
			o.Expect(err).ShouldNot(o.HaveOccurred())
			o.Expect(podNames).Should(o.HaveLen(1))

			g.By("expecting the mongodb service is answering for ping")
			mongo := db.NewMongoDB(podNames[0], "")

			for times := 0; times < 10; times++ {
				ok, err := mongo.IsReady(oc)
				if ok {
					break
				}

				if times == 10 {
					o.Expect(err).ShouldNot(o.HaveOccurred())
					o.Expect(ok).Should(o.BeTrue())
					break
				}

				time.Sleep(1 * time.Second)
			}

			g.By("expecting that we can insert a new record")
			result, err := mongo.Query(oc, `db.foo.save({ "status": "passed" })`)
			o.Expect(err).ShouldNot(o.HaveOccurred())
			o.Expect(result).Should(o.ContainSubstring(`WriteResult({ "nInserted" : 1 })`))

			g.By("expecting that we can read a record")
			findCmd := "printjson(db.foo.find({}, {_id: 0}).toArray())" // don't include _id field to output because it changes every time
			result, err = mongo.Query(oc, findCmd)
			o.Expect(err).ShouldNot(o.HaveOccurred())
			o.Expect(result).Should(o.ContainSubstring(`{ "status" : "passed" }`))
		})
	})

})
