// Package mutate deals with AdmissionReview requests and responses, it takes in the request body and returns a readily converted JSON []byte that can be
// returned from a http Handler w/o needing to further convert or modify it, it also makes testing Mutate() kind of easy w/o need for a fake http server, etc.
package mutate

import (
	"encoding/json"
	"fmt"
	"log"

	v1beta1 "k8s.io/api/admission/v1beta1"
  // batch types: https://github.com/kubernetes/api/blob/master/batch/v1/types.go
  batchv1 "k8s.io/api/batch/v1"
  corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Mutate mutates
func Mutate(body []byte, verbose bool) ([]byte, error) {
	if verbose {
		log.Printf("recv: %s\n", string(body)) // untested section
	}

	// unmarshal request into AdmissionReview struct
	admReview := v1beta1.AdmissionReview{}
	if err := json.Unmarshal(body, &admReview); err != nil {
		return nil, fmt.Errorf("unmarshaling request failed with %s", err)
	}

	var err error
	//var pod *corev1.Pod
  var job *batchv1.Job

	responseBody := []byte{}
	ar := admReview.Request
	resp := v1beta1.AdmissionResponse{}

	if ar != nil {

    // jobs: https://kubernetes.io/docs/concepts/workloads/controllers/job/

		// get the job object and unmarshal it into its struct, if we cannot, we might as well stop here
		if err := json.Unmarshal(ar.Object.Raw, &job); err != nil {
			return nil, fmt.Errorf("unable unmarshal job json object %v", err)
		}
		// set response options
		resp.Allowed = true
		resp.UID = ar.UID
		pT := v1beta1.PatchTypeJSONPatch
		resp.PatchType = &pT // it's annoying that this needs to be a pointer as you cannot give a pointer to a constant?

		// add some audit annotations, helpful to know why a object was modified, maybe (?)
		resp.AuditAnnotations = map[string]string{
			"mutateme": "yup it did it",
		}

    // generateName: https://kubernetes.io/docs/reference/using-api/api-concepts/#generated-values

    //TODO: check to see if the job metadata already has a generateName field
    // if so then we can noOP
    // if it does not have a generateName field but does have a name field then
    // we copy name to generateName and remove name
    // if it does not have name or generateName we noOp and let k8s deal with it

		// the actual mutation is done by a string in JSONPatch style, i.e. we don't _actually_ modify the object, but
		// tell K8S how it should modifiy it

    if job.Metadata.GenerateName == nil && job.Metadata.Name != nil {
		  p := []map[string]string{}
		  patch := map[string]string{
		 	  "op":    "add",
		 	  "path":  "/metadata/generateName",
		 	  "value": job.Metadata.Name,
		  }
		  p = append(p, patch)
		  // parse the []map into JSON
		  resp.Patch, err = json.Marshal(p)

		  // Success, of course ;)
		  resp.Result = &metav1.Status{
		    Status: "Success",
		  }
    }

		admReview.Response = &resp
		// back into JSON so we can return the finished AdmissionReview w/ Response directly
		// w/o needing to convert things in the http handler
		responseBody, err = json.Marshal(admReview)
		if err != nil {
			return nil, err // untested section
		}
	}

	if verbose {
		log.Printf("resp: %s\n", string(responseBody)) // untested section
	}

	return responseBody, nil
}
