package types

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/koki/short/util"
)

type PersistentVolumeWrapper struct {
	PersistentVolume PersistentVolume `json:"persistent_volume"`
}

type PersistentVolume struct {
	PersistentVolumeMeta
	PersistentVolumeSource
}

type PersistentVolumeMeta struct {
	Version     string            `json:"version,omitempty"`
	Cluster     string            `json:"cluster,omitempty"`
	Name        string            `json:"name,omitempty"`
	Namespace   string            `json:"namespace,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`

	Storage       *resource.Quantity            `json:"storage,omitempty"`
	AccessModes   *AccessModes                  `json:"modes,omitempty"`
	Claim         *v1.ObjectReference           `json:"claim,omitempty"`
	ReclaimPolicy PersistentVolumeReclaimPolicy `json:"reclaim,omitempty"`
	StorageClass  string                        `json:"storage_class,omitempty"`

	// comma-separated list of options
	MountOptions string `json:"mount_options,omitempty" protobuf:"bytes,7,opt,name=mountOptions"`

	Status *v1.PersistentVolumeStatus `json:"status,omitempty"`
}

type PersistentVolumeReclaimPolicy string

const (
	PersistentVolumeReclaimRecycle PersistentVolumeReclaimPolicy = "recycle"
	PersistentVolumeReclaimDelete  PersistentVolumeReclaimPolicy = "delete"
	PersistentVolumeReclaimRetain  PersistentVolumeReclaimPolicy = "retain"
)

type PersistentVolumeSource struct {
	GcePD        *GcePDVolume
	AwsEBS       *AwsEBSVolume
	HostPath     *HostPathVolume
	Glusterfs    *GlusterfsVolume
	NFS          *NFSVolume
	ISCSI        *ISCSIVolume
	Cinder       *CinderVolume
	FibreChannel *FibreChannelVolume
	Flocker      *FlockerVolume
	Flex         *FlexVolume
	Vsphere      *VsphereVolume
	Quobyte      *QuobyteVolume
	AzureDisk    *AzureDiskVolume
	PhotonPD     *PhotonPDVolume
	Portworx     *PortworxVolume
	RBD          *RBDPersistentVolume
}

const (
	VolumeTypeLocal = "local"
)

type RBDPersistentVolume struct {
	CephMonitors []string         `json:"monitors"`
	RBDImage     string           `json:"image"`
	FSType       string           `json:"fs,omitempty"`
	RBDPool      string           `json:"pool,omitempty"`
	RadosUser    string           `json:"user,omitempty"`
	Keyring      string           `json:"keyring,omitempty"`
	SecretRef    *SecretReference `json:"secret,omitempty"`
	ReadOnly     bool             `json:"ro,omitempty"`
}

type SecretReference struct {
	Name      string `json:"-"`
	Namespace string `json:"-"`
}

// comma-separated list of modes
type AccessModes struct {
	Modes []v1.PersistentVolumeAccessMode
}

func (a *AccessModes) ToString() (string, error) {
	if a == nil {
		return "", nil
	}

	if len(a.Modes) == 0 {
		return "", nil
	}

	modes := make([]string, len(a.Modes))
	for i, mode := range a.Modes {
		switch mode {
		case v1.ReadOnlyMany:
			modes[i] = "ro"
		case v1.ReadWriteMany:
			modes[i] = "rw"
		case v1.ReadWriteOnce:
			modes[i] = "rw-once"
		default:
			return "", util.InvalidInstanceError(mode)
		}
	}

	return strings.Join(modes, ","), nil
}

func (a *AccessModes) InitFromString(s string) error {
	modes := strings.Split(s, ",")
	if len(modes) == 0 {
		a.Modes = nil
		return nil
	}

	a.Modes = make([]v1.PersistentVolumeAccessMode, len(modes))
	for i, mode := range modes {
		switch mode {
		case "ro":
			a.Modes[i] = v1.ReadOnlyMany
		case "rw":
			a.Modes[i] = v1.ReadWriteMany
		case "rw-once":
			a.Modes[i] = v1.ReadWriteOnce
		default:
			return util.InvalidValueErrorf(a, "couldn't parse (%s)", s)
		}
	}

	return nil
}

func (a AccessModes) MarshalJSON() ([]byte, error) {
	str, err := a.ToString()
	if err != nil {
		return nil, err
	}

	b, err := json.Marshal(&str)
	if err != nil {
		return nil, util.InvalidInstanceErrorf(a, "couldn't marshal to JSON: %s", err.Error())
	}

	return b, nil
}

func (a *AccessModes) UnmarshalJSON(data []byte) error {
	str := ""
	err := json.Unmarshal(data, &str)
	if err != nil {
		return util.InvalidValueForTypeErrorf(string(data), a, "couldn't unmarshal from JSON: %s", err.Error())
	}

	return a.InitFromString(str)
}

func (v *PersistentVolume) UnmarshalJSON(data []byte) error {
	err := json.Unmarshal(data, &v.PersistentVolumeSource)
	if err != nil {
		return util.InvalidValueForTypeErrorf(string(data), v, "couldn't unmarshal volume source from JSON: %s", err.Error())
	}

	err = json.Unmarshal(data, &v.PersistentVolumeMeta)
	if err != nil {
		return util.InvalidValueForTypeErrorf(string(data), v, "couldn't unmarshal metadata from JSON: %s", err.Error())
	}

	return nil
}

func (v PersistentVolume) MarshalJSON() ([]byte, error) {
	b, err := json.Marshal(v.PersistentVolumeMeta)
	if err != nil {
		return nil, util.InvalidInstanceErrorf(v, "couldn't marshal metadata to JSON: %s", err.Error())
	}

	bb, err := json.Marshal(v.PersistentVolumeSource)
	if err != nil {
		return nil, err
	}

	metaObj := map[string]interface{}{}
	err = json.Unmarshal(b, &metaObj)
	if err != nil {
		return nil, util.InvalidValueForTypeErrorf(string(b), v.PersistentVolumeMeta, "couldn't convert metadata to dictionary: %s", err.Error())
	}

	sourceObj := map[string]interface{}{}
	err = json.Unmarshal(bb, &sourceObj)
	if err != nil {
		return nil, util.InvalidValueForTypeErrorf(string(bb), v.PersistentVolumeSource, "couldn't convert volume source to dictionary: %s", err.Error())
	}

	// Merge metadata with volume-source
	for key, val := range metaObj {
		sourceObj[key] = val
	}

	result, err := json.Marshal(sourceObj)
	if err != nil {
		return nil, util.InvalidValueForTypeErrorf(result, v, "couldn't marshal PersistentVolume-as-dictionary to JSON: %s", err.Error())
	}

	return result, nil
}

func (v *PersistentVolumeSource) UnmarshalJSON(data []byte) error {
	var err error
	obj := map[string]interface{}{}
	err = json.Unmarshal(data, &obj)
	if err != nil {
		return util.InvalidValueErrorf(string(data), "expected dictionary for persistent volume")
	}

	var selector []string
	if val, ok := obj["vol_id"]; ok {
		if volName, ok := val.(string); ok {
			selector = strings.Split(volName, ":")
		} else {
			return util.InvalidValueErrorf(string(data), "expected string for key \"vol_id\"")
		}
	}

	volType, err := util.GetStringEntry(obj, "vol_type")
	if err != nil {
		return err
	}

	return v.Unmarshal(obj, volType, selector)
}

func (v *PersistentVolumeSource) Unmarshal(obj map[string]interface{}, volType string, selector []string) error {
	switch volType {
	case VolumeTypeGcePD:
		v.GcePD = &GcePDVolume{}
		return v.GcePD.Unmarshal(obj, selector)
	case VolumeTypeAwsEBS:
		v.AwsEBS = &AwsEBSVolume{}
		return v.AwsEBS.Unmarshal(obj, selector)
	case VolumeTypeHostPath:
		v.HostPath = &HostPathVolume{}
		return v.HostPath.Unmarshal(selector)
	case VolumeTypeGlusterfs:
		v.Glusterfs = &GlusterfsVolume{}
		return v.Glusterfs.Unmarshal(obj, selector)
	case VolumeTypeNFS:
		v.NFS = &NFSVolume{}
		return v.NFS.Unmarshal(selector)
	case VolumeTypeISCSI:
		v.ISCSI = &ISCSIVolume{}
		return v.ISCSI.Unmarshal(obj, selector)
	case VolumeTypeCinder:
		v.Cinder = &CinderVolume{}
		return v.Cinder.Unmarshal(obj, selector)
	case VolumeTypeFibreChannel:
		v.FibreChannel = &FibreChannelVolume{}
		return v.FibreChannel.Unmarshal(obj, selector)
	case VolumeTypeFlocker:
		v.Flocker = &FlockerVolume{}
		return v.Flocker.Unmarshal(selector)
	case VolumeTypeFlex:
		v.Flex = &FlexVolume{}
		return v.Flex.Unmarshal(obj, selector)
	case VolumeTypeVsphere:
		v.Vsphere = &VsphereVolume{}
		return v.Vsphere.Unmarshal(obj, selector)
	case VolumeTypeQuobyte:
		v.Quobyte = &QuobyteVolume{}
		return v.Quobyte.Unmarshal(obj, selector)
	case VolumeTypeAzureDisk:
		v.AzureDisk = &AzureDiskVolume{}
		return v.AzureDisk.Unmarshal(obj, selector)
	case VolumeTypePhotonPD:
		v.PhotonPD = &PhotonPDVolume{}
		return v.PhotonPD.Unmarshal(selector)
	case VolumeTypePortworx:
		v.Portworx = &PortworxVolume{}
		return v.Portworx.Unmarshal(obj, selector)
	case VolumeTypeRBD:
		v.RBD = &RBDPersistentVolume{}
		return v.RBD.Unmarshal(obj, selector)
	default:
		return util.InvalidValueErrorf(volType, "unsupported volume type (%s)", volType)
	}
}

func (v PersistentVolumeSource) MarshalJSON() ([]byte, error) {
	var marshalledVolume *MarshalledVolume
	var err error
	if v.GcePD != nil {
		marshalledVolume, err = v.GcePD.Marshal()
	}
	if v.AwsEBS != nil {
		marshalledVolume, err = v.AwsEBS.Marshal()
	}
	if v.HostPath != nil {
		marshalledVolume, err = v.HostPath.Marshal()
	}
	if v.Glusterfs != nil {
		marshalledVolume, err = v.Glusterfs.Marshal()
	}
	if v.NFS != nil {
		marshalledVolume, err = v.NFS.Marshal()
	}
	if v.ISCSI != nil {
		marshalledVolume, err = v.ISCSI.Marshal()
	}
	if v.Cinder != nil {
		marshalledVolume, err = v.Cinder.Marshal()
	}
	if v.FibreChannel != nil {
		marshalledVolume, err = v.FibreChannel.Marshal()
	}
	if v.Flocker != nil {
		marshalledVolume, err = v.Flocker.Marshal()
	}
	if v.Flex != nil {
		marshalledVolume, err = v.Flex.Marshal()
	}
	if v.Vsphere != nil {
		marshalledVolume, err = v.Vsphere.Marshal()
	}
	if v.Quobyte != nil {
		marshalledVolume, err = v.Quobyte.Marshal()
	}
	if v.AzureDisk != nil {
		marshalledVolume, err = v.AzureDisk.Marshal()
	}
	if v.PhotonPD != nil {
		marshalledVolume, err = v.PhotonPD.Marshal()
	}
	if v.Portworx != nil {
		marshalledVolume, err = v.Portworx.Marshal()
	}
	if v.RBD != nil {
		marshalledVolume, err = v.RBD.Marshal()
	}

	if err != nil {
		return nil, err
	}

	if marshalledVolume == nil {
		return nil, util.InvalidInstanceErrorf(v, "empty volume definition")
	}

	if len(marshalledVolume.ExtraFields) == 0 {
		marshalledVolume.ExtraFields = map[string]interface{}{}
	}

	obj := marshalledVolume.ExtraFields
	obj["vol_type"] = marshalledVolume.Type
	if len(marshalledVolume.Selector) > 0 {
		obj["vol_id"] = strings.Join(marshalledVolume.Selector, ":")
	}

	return json.Marshal(obj)
}

var secretRefRegexp = regexp.MustCompile(`^(.*):([^:]*)`)

func (s *SecretReference) UnmarshalJSON(data []byte) error {
	str := ""
	err := json.Unmarshal(data, &str)
	if err != nil {
		return util.ContextualizeErrorf(err, "secret ref should be a string")
	}

	matches := secretRefRegexp.FindStringSubmatch(str)
	if len(matches) > 0 {
		s.Namespace = matches[1]
		s.Name = matches[2]
	} else {
		s.Name = str
	}

	return nil
}

func (s SecretReference) MarshalJSON() ([]byte, error) {
	if len(s.Namespace) > 0 {
		return json.Marshal(fmt.Sprintf("%s:%s", s.Namespace, s.Name))
	}

	return json.Marshal(s.Name)
}

func (s *RBDPersistentVolume) Unmarshal(obj map[string]interface{}, selector []string) error {
	if len(selector) != 0 {
		return util.InvalidValueErrorf(selector, "expected zero selector segments for %s", VolumeTypeRBD)
	}

	err := util.UnmarshalMap(obj, &s)
	if err != nil {
		return util.ContextualizeErrorf(err, VolumeTypeRBD)
	}

	return nil
}

func (s RBDPersistentVolume) Marshal() (*MarshalledVolume, error) {
	obj, err := util.MarshalMap(&s)
	if err != nil {
		return nil, util.ContextualizeErrorf(err, VolumeTypeRBD)
	}

	return &MarshalledVolume{
		Type:        VolumeTypeRBD,
		ExtraFields: obj,
	}, nil
}
