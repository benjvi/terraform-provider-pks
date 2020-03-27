package pks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

type Token struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
	Scope       string `json:"scope"`
	Jti         string `json:"jti"`
}

type ClusterRequest struct {
	Name       string            `json:"name"`
	PlanName   string            `json:"plan_name"`
	Parameters ClusterParameters `json:"parameters"`
	NetworkProfileName ClusterParameters `json:"network_profile_name"`
}

type ClusterParameters struct {
	KubernetesMasterHost      string `json:"kubernetes_master_host"`
	KubernetesMasterPort      int64  `json:"kubernetes_master_port,omitempty"`
	KubernetesWorkerInstances int64  `json:"kubernetes_worker_instances,omitempty"`
}

type ClusterResponse struct {
	Name                  string            `json:"name"`
	PlanName              string            `json:"plan_name"`
	LastAction            string            `json:"last_action"`
	LastActionState       string            `json:"last_action_state"`
	LastActionDescription string            `json:"last_action_description"`
	Uuid                  string            `json:"uuid"`
	K8sVersion            string            `json:"k8s_version"`
	PksVersion            string            `json:"pks_version"`
	KubernetesMasterIps   []string          `json:"kubernetes_master_ips"`
	Parameters            ClusterParameters `json:"parameters"`
	NetworkProfileName    string `json:"network_profile_name"`
}

type UpdateClusterParameters struct {
	KubernetesWorkerInstances int64 `json:"kubernetes_worker_instances,omitempty"`
}

func ClientLogin(httpClient *http.Client, hostname, clientId, clientSecret string) (string, error) {
	/*
		Replicating this working curl command:
		curl -s https://${PKS_ADDRESS}:8443/oauth/token
		     -k -X POST -H 'Accept: application/json;charset=utf-8'
		     -u "client_id:client_secret" -H 'Content-Type: application/x-www-form-urlencoded;charset=utf-8'
		     -d 'grant_type=client_credentials'
	*/
	tokenReqData := "grant_type=client_credentials"
	req, _ := http.NewRequest("POST", "https://"+hostname+":8443/oauth/token", strings.NewReader(tokenReqData))
	req.SetBasicAuth(clientId, clientSecret)
	req.Header["Content-Type"] = []string{"application/x-www-form-urlencoded;charset=utf-8"}
	req.Header["Accept"] = []string{"application/json; charset=utf-8"}

	resp, err := httpClient.Do(req)
	if err != nil {
		// this doesn't catch 4xx/5xx !
		return "", fmt.Errorf("error connecting to PKS API to get token %q: %q", req.URL.String(), err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		body, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("PKS token request returned unexpected status %q with response: %q", resp.Status, body)
	}

	var token Token
	err = json.NewDecoder(resp.Body).Decode(&token)
	if err != nil {
		return "", fmt.Errorf("error parsing token response from PKS API %q: %q", req.URL.String(), err.Error())
	}

	return token.AccessToken, nil
}

func GetCluster(client *Client, clusterName string) (*ClusterResponse, bool, error) {
	req, _ := http.NewRequest("GET", "https://"+client.hostname+":9021/v1/clusters/"+clusterName, nil)
	req.Header["Authorization"] = []string{"Bearer " + client.token}
	req.Header["Accept"] = []string{"application/json; charset=utf-8"}
	resp, err := client.httpClient.Do(req)
	if err != nil {
		// this doesn't catch 4xx/5xx !
		return nil, false, fmt.Errorf("error reading cluster from PKS API %q: %q", req.URL.String(), err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, false, nil
	} else if resp.StatusCode > 299 {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, false, fmt.Errorf("cluster read returned unexpected status %q with response: %q", resp.Status, body)
	}

	var cr ClusterResponse
	err = json.NewDecoder(resp.Body).Decode(&cr)
	if err != nil {
		return nil, false, fmt.Errorf("error parsing cluster response from PKS API %q: %q", req.URL.String(), err.Error())
	}
	return &cr, true, nil
}

func CreateCluster(client *Client, clusterReq ClusterRequest) error {
	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(clusterReq)
	req, _ := http.NewRequest("POST", "https://"+client.hostname+":9021/v1/clusters", b)
	req.Header["Authorization"] = []string{"Bearer " + client.token}
	req.Header["Content-Type"] = []string{"application/json; charset=utf-8"}
	req.Header["Accept"] = []string{"application/json; charset=utf-8"}
	resp, err := client.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("POST to API to create cluster failed: %q", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("cluster creation returned unexpected status %q with response: %q", resp.Status, body)
	}
	return nil
}

func UpdateCluster(client *Client, clusterName string, updateClusterReq UpdateClusterParameters) error {
	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(updateClusterReq)
	req, _ := http.NewRequest("PATCH", "https://"+client.hostname+":9021/v1/clusters/"+clusterName, b)
	req.Header["Authorization"] = []string{"Bearer " + client.token}
	req.Header["Content-Type"] = []string{"application/json; charset=utf-8"}
	req.Header["Accept"] = []string{"application/json; charset=utf-8"}
	resp, err := client.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("POST to API to create cluster failed: %q", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("cluster creation returned unexpected status %q with response: %q", resp.Status, body)
	}
	return nil
}

func DeleteCluster(client *Client, clusterName string) error {
	req, _ := http.NewRequest("DELETE", "https://"+client.hostname+":9021/v1/clusters/"+clusterName, nil)
	req.Header["Authorization"] = []string{"Bearer " + client.token}
	req.Header["Accept"] = []string{"application/json; charset=utf-8"}

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error deleting cluster from PKS API %q: %q", req.URL.String(), err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		// cluster was already deleted
		return nil
	} else if resp.StatusCode > 299 {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("Cluster delete returned unexpected status %q with response: %q", resp.Status, body)
	}

	return nil
}

func WaitForClusterAction(client *Client, clusterName, action string) error {
	timeout := time.After(time.Duration(client.maxWaitMin) * time.Minute)
	tick := time.Tick(time.Duration(client.waitPollIntervalSec) * time.Second)

	// may take a few moments for our action to be registered in PKS
	maxPollingRetries := 3
	pollingRetries := 0

	// Keep trying until we're timed out or got a result or got an error
	for {
		select {
		case <-timeout:
			return fmt.Errorf("Timed out waiting for action %q to succeed on cluster %q", action, clusterName)
		case <-tick:
			cr, exists, err := GetCluster(client, clusterName)
			if err != nil {
				return err
			}

			// checking if cluster exists
			if strings.EqualFold("DELETE", action) && !exists {
				// delete action completed ok
				return nil
			} else if !exists {
				if pollingRetries < maxPollingRetries {
					pollingRetries = pollingRetries + 1
					break
				} else {
					return fmt.Errorf("Cluster %q not found while waiting for action %q", clusterName, action)
				}
			}

			// checking the action is what we expected
			if !strings.EqualFold(cr.LastAction, action) {
				if pollingRetries < maxPollingRetries {
					pollingRetries = pollingRetries + 1
					break
				} else {
					return fmt.Errorf("Found an unexpected action on our cluster: %q, status: %q (%q)", cr.LastAction,
						cr.LastActionState, cr.LastActionDescription)
				}
			}

			// check the status of our action
			if strings.EqualFold(cr.LastActionState, "in progress") {
				break
			} else if strings.EqualFold(cr.LastActionState, "failed") {
				return fmt.Errorf("Cluster creation failed with error: %q", cr.LastActionDescription)
			} else if strings.EqualFold(cr.LastActionState, "succeeded") {
				return nil
			} else {
				return fmt.Errorf("Unexpected cluster status: %q", cr.LastActionState)
			}
		}
	}
}
