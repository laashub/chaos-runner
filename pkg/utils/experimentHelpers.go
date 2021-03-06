package utils

import (
	"os"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

func CreateExperimentList(engineDetails *EngineDetails) []ExperimentDetails {
	var ExperimentDetailsList []ExperimentDetails
	for i, _ := range engineDetails.Experiments {
		ExperimentDetailsList = append(ExperimentDetailsList, NewExperimentDetails(engineDetails, i))
	}
	return ExperimentDetailsList
}

//SetValueFromChaosExperiment sets value in experimentDetails struct from chaosExperiment
func (expDetails *ExperimentDetails) SetValueFromChaosExperiment(clients ClientSets, engine *EngineDetails) error {
	if err := expDetails.SetImage(clients); err != nil {
		return err
	}
	if err := expDetails.SetArgs(clients); err != nil {
		return err
	}
	if err := expDetails.SetLabels(engine, clients); err != nil {
		return err
	}
	return nil
}

//SetENV sets ENV values in experimentDetails struct.
func (expDetails *ExperimentDetails) SetENV(engineDetails EngineDetails, clients ClientSets) error {
	// Get the Default ENV's from ChaosExperiment
	klog.V(0).Infof("Getting the ENV Variables")
	if err := expDetails.SetDefaultEnv(clients); err != nil {
		return err
	}

	// OverWriting the Defaults Varibles from the ChaosEngine ENV
	if err := expDetails.SetEnvFromEngine(engineDetails.Name, clients); err != nil {
		return err
	}
	// Store ENV in a map
	ENVList := map[string]string{"CHAOSENGINE": engineDetails.Name, "APP_LABEL": engineDetails.AppLabel, "CHAOS_NAMESPACE": engineDetails.EngineNamespace, "APP_NAMESPACE": os.Getenv("APP_NAMESPACE"), "APP_KIND": engineDetails.AppKind, "AUXILIARY_APPINFO": engineDetails.AuxiliaryAppInfo, "CHAOS_UID": engineDetails.UID}
	// Adding some addition ENV's from spec.AppInfo of ChaosEngine
	for key, value := range ENVList {
		expDetails.Env[key] = value
	}
	return nil
}

// NewExperimentDetails initilizes the structure
func NewExperimentDetails(engineDetails *EngineDetails, i int) ExperimentDetails {
	var experimentDetails ExperimentDetails
	experimentDetails.Env = make(map[string]string)
	experimentDetails.ExpLabels = make(map[string]string)

	// Initial set to values from EngineDetails Struct
	experimentDetails.Name = engineDetails.Experiments[i]
	experimentDetails.SvcAccount = engineDetails.SvcAccount
	experimentDetails.Namespace = os.Getenv("CHAOS_NAMESPACE")

	// Generation of Random String for appending it into Job Name
	randomString := RandomString()
	// Setting the JobName in Experiment Realted struct
	experimentDetails.JobName = experimentDetails.Name + "-" + randomString
	return experimentDetails
}

// HandleChaosExperimentExistence will check the experiment in the app namespace
func (expDetails *ExperimentDetails) HandleChaosExperimentExistence(engineDetails EngineDetails, clients ClientSets) error {

	_, err := clients.LitmusClient.LitmuschaosV1alpha1().ChaosExperiments(expDetails.Namespace).Get(expDetails.Name, metav1.GetOptions{})
	if err != nil {
		if err := engineDetails.ExperimentNotFoundPatchEngine(expDetails, clients); err != nil {
			return errors.Wrapf(err, "Unable to patch Chaos Engine Name: %v, in namespace: %v, due to error: %v", engineDetails.Name, engineDetails.EngineNamespace, err)
		}
		return errors.Wrapf(err, "Unable to list Chaos Experiment Name: %v,in Namespace: %v, due to error: %v", expDetails.Name, expDetails.Namespace, err)
	}

	return nil
}

// SetDefaultEnv sets the Env's in Experiment Structure
func (expDetails *ExperimentDetails) SetDefaultEnv(clients ClientSets) error {
	experimentEnv, err := clients.LitmusClient.LitmuschaosV1alpha1().ChaosExperiments(expDetails.Namespace).Get(expDetails.Name, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "Unable to get the Default ENV from ChaosExperiment, error : %v", err)
	}

	envList := experimentEnv.Spec.Definition.ENVList
	expDetails.Env = make(map[string]string)
	for i := range envList {
		key := envList[i].Name
		value := envList[i].Value
		expDetails.Env[key] = value
	}
	return nil
}

// SetEnvFromEngine will over-ride the default variables from one provided in the chaosEngine
func (expDetails *ExperimentDetails) SetEnvFromEngine(engineName string, clients ClientSets) error {

	engineSpec, err := clients.LitmusClient.LitmuschaosV1alpha1().ChaosEngines(expDetails.Namespace).Get(engineName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "Unable to get ChaosEngine Resource in namespace: %v", expDetails.Namespace)
	}
	envList := engineSpec.Spec.Experiments
	for i := range envList {
		if envList[i].Name == expDetails.Name {
			keyValue := envList[i].Spec.Components.ENV
			for j := range keyValue {
				expDetails.Env[keyValue[j].Name] = keyValue[j].Value
			}
		}
	}
	return nil
}

// SetLabels sets the Experiment Labels, in Experiment Structure
func (expDetails *ExperimentDetails) SetLabels(engine *EngineDetails, clients ClientSets) error {
	expirementSpec, err := clients.LitmusClient.LitmuschaosV1alpha1().ChaosExperiments(expDetails.Namespace).Get(expDetails.Name, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "Unable to get ChaosExperiment instance in namespace: %v", expDetails.Namespace)
	}
	expDetails.ExpLabels = expirementSpec.Spec.Definition.Labels
	expDetails.ExpLabels["chaosUID"] = engine.UID
	return nil
}

// SetImage sets the Experiment Image, in Experiment Structure
func (expDetails *ExperimentDetails) SetImage(clients ClientSets) error {
	expirementSpec, err := clients.LitmusClient.LitmuschaosV1alpha1().ChaosExperiments(expDetails.Namespace).Get(expDetails.Name, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "Unable to get ChaosExperiment instance in namespace: %v", expDetails.Namespace)
	}
	expDetails.ExpImage = expirementSpec.Spec.Definition.Image
	return nil
}

// SetArgs sets the Experiment Image, in Experiment Structure
func (expDetails *ExperimentDetails) SetArgs(clients ClientSets) error {
	expirementSpec, err := clients.LitmusClient.LitmuschaosV1alpha1().ChaosExperiments(expDetails.Namespace).Get(expDetails.Name, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "Unable to get ChaosExperiment instance in namespace: %v", expDetails.Namespace)
	}
	expDetails.ExpArgs = expirementSpec.Spec.Definition.Args
	return nil
}

// SetValueFromChaosResources fetchs required values from various Chaos Resources
func (expDetails *ExperimentDetails) SetValueFromChaosResources(engineDetails *EngineDetails, clients ClientSets) error {
	if err := expDetails.SetValueFromChaosEngine(engineDetails, clients); err != nil {
		return errors.Wrapf(err, "Unable to set value from Chaos Engine due to error: %v", err)
	}

	if err := engineDetails.SetValueFromChaosRunner(clients); err != nil {
		return errors.Wrapf(err, "Unable to set value from Chaos Runner due to error: %v", err)
	}
	if err := expDetails.HandleChaosExperimentExistence(*engineDetails, clients); err != nil {
		return errors.Wrapf(err, "Unable to get ChaosExperiment Name: %v, in namespace: %v, due to error: %v", expDetails.Name, expDetails.Namespace, err)
	}
	if err := expDetails.SetValueFromChaosExperiment(clients, engineDetails); err != nil {
		return errors.Wrapf(err, "Unable to set value from Chaos Experiment due to error: %v", err)
	}
	return nil
}

// SetValueFromChaosRunner fetch the engineUID from ChaosRunner
func (engine *EngineDetails) SetValueFromChaosRunner(clients ClientSets) error {
	runnerName := engine.Name + "-runner"
	runnerSpec, err := clients.KubeClient.CoreV1().Pods(engine.EngineNamespace).Get(runnerName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "Unable to get runner pod in namespace: %v", engine.EngineNamespace)
	}
	chaosUID := runnerSpec.Labels["chaosUID"]
	if chaosUID != "" {
		engine.UID = chaosUID
	} else {
		return errors.Wrapf(err, "Unable to get ChaosEngine UID, due to error: %v", err)
	}
	return nil
}

func (expDetails *ExperimentDetails) SetValueFromChaosEngine(engine *EngineDetails, clients ClientSets) error {

	chaosEngine, err := engine.GetChaosEngine(clients)
	if err != nil {
		return errors.Wrapf(err, "Unable to get chaosEngine in namespace: %s", engine.EngineNamespace)
	}
	expDetails.Namespace = chaosEngine.Namespace
	return nil
}
