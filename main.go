package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	configpb "github.com/cloudprober/cloudprober/config/proto"
	httppb "github.com/cloudprober/cloudprober/probes/http/proto"
	probespb "github.com/cloudprober/cloudprober/probes/proto"
	targetspb "github.com/cloudprober/cloudprober/targets/proto"

	backstage "github.com/tdabasinskas/go-backstage/v2/backstage"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

var namespace = flag.String("namespace", "github.com/nrhtr/probegen", "Namespace with which to prefix annotation keys (e.g. example.com)")
var backstageUrl = flag.String("backstage-url", "", "URL for backstage, e.g. https://backstage.ops.example.com/")
var pretty = flag.Bool("pretty", false, "whether or not to format the generated config")

func main() {

	flag.Parse()

	if *backstageUrl == "" {
		flag.Usage()
		os.Exit(1)
	}

	err := generateProbeDefinitions()
	if err != nil {
		log.Fatalf("error generating probe definitions: %v", err)
	}
}

func generateProbeDefinitions() error {

	httpClient := &http.Client{}

	c, err := backstage.NewClient(*backstageUrl, "default", httpClient)
	if err != nil {
		log.Fatalf("Unable to initialise backstage client: %v", err)
	}

	entities, _, err := c.Catalog.Entities.List(context.Background(), &backstage.ListEntityOptions{
		Filters: []string{
			"kind=component",
		},
	})
	if err != nil {
		return fmt.Errorf("error retrieving components from Backstage: %v", err)
	}

	log.Printf("found %d components in backstage", len(entities))

	probeDefs, err := generateProbes(entities)
	if err != nil {
		return fmt.Errorf("error generating probe definitions: %v", err)
	}

	err = writeProbeDefinitions(probeDefs)
	if err != nil {
		return fmt.Errorf("error saving probe definitions: %v", err)
	}

	return nil
}

func scopedKey(s string) string {
	return fmt.Sprintf("%s/%s", *namespace, s)
}

func generateProbes(components []backstage.Entity) ([]*probespb.ProbeDef, error) {
	var probeDefs []*probespb.ProbeDef

	for _, component := range components {
		isConfigured := false

		name := component.Metadata.Name
		kind := component.Kind

		probeType := "HTTP"
		probeTargets := ""
		probeHttpProtocol := httppb.ProbeConf_HTTPS
		probeHttpRelativeUrl := "/"
		probeHttpMethod := httppb.ProbeConf_GET
		probeHttpBody := []string{}
		//probeHttpValidator := nil
		probeInterval := "10s"

		log.Printf("checking component '%s' for annotations", name)

		for k, v := range component.Metadata.Annotations {
			log.Printf("annotation: %s=%s\n", k, v)
			if strings.HasPrefix(k, fmt.Sprintf("%s/probe", *namespace)) {
				isConfigured = true
			}
			switch k {
			case scopedKey("probe-type"):
				{
					probeType = v
				}
			case scopedKey("probe-targets"):
				{
					probeTargets = v
				}
			case scopedKey("probe-interval"):
				{
					probeInterval = v
				}
			case scopedKey("probe-http-protocol"):
				{
					if v == "HTTPS" {
						probeHttpProtocol = httppb.ProbeConf_HTTPS
					} else if v == "HTTP" {
						probeHttpProtocol = httppb.ProbeConf_HTTP
					}
				}
				{
					probeHttpRelativeUrl = v
				}
			case scopedKey("probe-http-method"):
				{
					switch v {
					case "GET":
						{
							probeHttpMethod = httppb.ProbeConf_GET
						}
					case "POST":
						{
							probeHttpMethod = httppb.ProbeConf_POST

						}
					}
				}
			case scopedKey("probe-http-relative-url"):
				{
					probeHttpRelativeUrl = v
				}
			case scopedKey("probe-http-body"):
				{
					probeHttpBody = append(probeHttpBody, v)
				}
			}
		}

		if !isConfigured {
			continue
		}

		log.Printf("defining probe for %s (%s) of type %s\n", name, kind, probeType)

		probeDef := &probespb.ProbeDef{
			Name: proto.String(fmt.Sprintf("probe-%s", component.Metadata.Name)),
			Type: probespb.ProbeDef_HTTP.Enum(),

			Targets: &targetspb.TargetsDef{
				Type: &targetspb.TargetsDef_HostNames{
					HostNames: probeTargets,
				},
			},
			Interval: &probeInterval,
			Probe: &probespb.ProbeDef_HttpProbe{
				HttpProbe: &httppb.ProbeConf{
					Protocol:    &probeHttpProtocol,
					Body:        probeHttpBody,
					RelativeUrl: &probeHttpRelativeUrl,
					Method:      &probeHttpMethod,
				},
			},
		}

		probeDefs = append(probeDefs, probeDef)
	}

	return probeDefs, nil
}

func writeProbeDefinitions(probeDefs []*probespb.ProbeDef) error {
	probesConfig := &configpb.ProberConfig{
		Probe: probeDefs,
	}

	if *pretty {
		data := prototext.Format(probesConfig)
		fmt.Print(data)
	} else {
		data, err := prototext.Marshal(probesConfig)
		if err != nil {
			log.Fatalf("error marshalling protobuf config: %v", err)
		}
		os.Stdout.Write(data)
	}

	return nil
}
