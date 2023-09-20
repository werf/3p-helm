package resrc

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/samber/lo"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/werf/common"
	"helm.sh/helm/v3/pkg/werf/depnd"
	"helm.sh/helm/v3/pkg/werf/depnddetctr"
	"helm.sh/helm/v3/pkg/werf/utls"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/discovery"

	"github.com/werf/kubedog/pkg/trackers/rollout/multitrack"
)

type Type string

type ManageableBy string

const (
	ManageableByAnyone        ManageableBy = ""
	ManageableBySingleRelease ManageableBy = "manageable-by-single-release"
)

var annotationKeyHumanReleaseName = "meta.helm.sh/release-name"
var annotationKeyPatternReleaseName = regexp.MustCompile(`^meta.helm.sh/release-name$`)

var annotationKeyHumanReleaseNamespace = "meta.helm.sh/release-namespace"
var annotationKeyPatternReleaseNamespace = regexp.MustCompile(`^meta.helm.sh/release-namespace$`)

var labelKeyHumanManagedBy = "app.kubernetes.io/managed-by"
var labelKeyPatternManagedBy = regexp.MustCompile(`^app.kubernetes.io/managed-by$`)

var annotationKeyHumanHook = "helm.sh/hook"
var annotationKeyPatternHook = regexp.MustCompile(`^helm.sh/hook$`)

var annotationKeyHumanResourcePolicy = "helm.sh/resource-policy"
var annotationKeyPatternResourcePolicy = regexp.MustCompile(`^helm.sh/resource-policy$`)

var annotationKeyHumanDeletePolicy = "werf.io/delete-policy"
var annotationKeyPatternDeletePolicy = regexp.MustCompile(`^werf.io/delete-policy$`)

var annotationKeyHumanHookDeletePolicy = "helm.sh/hook-delete-policy"
var annotationKeyPatternHookDeletePolicy = regexp.MustCompile(`^helm.sh/hook-delete-policy$`)

var annotationKeyHumanReplicasOnCreation = "werf.io/replicas-on-creation"
var annotationKeyPatternReplicasOnCreation = regexp.MustCompile(`^werf.io/replicas-on-creation$`)

var annotationKeyHumanFailMode = "werf.io/fail-mode"
var annotationKeyPatternFailMode = regexp.MustCompile(`^werf.io/fail-mode$`)

var annotationKeyHumanFailuresAllowedPerReplica = "werf.io/failures-allowed-per-replica"
var annotationKeyPatternFailuresAllowedPerReplica = regexp.MustCompile(`^werf.io/failures-allowed-per-replica$`)

var annotationKeyHumanIgnoreReadinessProbeFailsFor = "werf.io/ignore-readiness-probe-fails-for-<container>"
var annotationKeyPatternIgnoreReadinessProbeFailsFor = regexp.MustCompile(`^werf.io/ignore-readiness-probe-fails-for-(?P<container>.+)$`)

var annotationKeyHumanLogRegex = "werf.io/log-regex"
var annotationKeyPatternLogRegex = regexp.MustCompile(`^werf.io/log-regex$`)

var annotationKeyHumanLogRegexFor = "werf.io/log-regex-for-<container>"
var annotationKeyPatternLogRegexFor = regexp.MustCompile(`^werf.io/log-regex-for-(?P<container>.+)$`)

var annotationKeyHumanNoActivityTimeout = "werf.io/no-activity-timeout"
var annotationKeyPatternNoActivityTimeout = regexp.MustCompile(`^werf.io/no-activity-timeout$`)

var annotationKeyHumanShowLogsOnlyForContainers = "werf.io/show-logs-only-for-containers"
var annotationKeyPatternShowLogsOnlyForContainers = regexp.MustCompile(`^werf.io/show-logs-only-for-containers$`)

var annotationKeyHumanShowServiceMessages = "werf.io/show-service-messages"
var annotationKeyPatternShowServiceMessages = regexp.MustCompile(`^werf.io/show-service-messages$`)

var annotationKeyHumanSkipLogs = "werf.io/skip-logs"
var annotationKeyPatternSkipLogs = regexp.MustCompile(`^werf.io/skip-logs$`)

var annotationKeyHumanSkipLogsForContainers = "werf.io/skip-logs-for-containers"
var annotationKeyPatternSkipLogsForContainers = regexp.MustCompile(`^werf.io/skip-logs-for-containers$`)

var annotationKeyHumanTrackTerminationMode = "werf.io/track-termination-mode"
var annotationKeyPatternTrackTerminationMode = regexp.MustCompile(`^werf.io/track-termination-mode$`)

var annotationKeyHumanWeight = "werf.io/weight"
var annotationKeyPatternWeight = regexp.MustCompile(`^werf.io/weight$`)

var annotationKeyHumanHookWeight = "helm.sh/hook-weight"
var annotationKeyPatternHookWeight = regexp.MustCompile(`^helm.sh/hook-weight$`)

var annotationKeyHumanDependency = "<name>.dependency.werf.io"
var annotationKeyPatternDependency = regexp.MustCompile(`^(?P<id>.+).dependency.werf.io$`)

var annotationKeyHumanExternalDependency = "<name>.external-dependency.werf.io"
var annotationKeyPatternExternalDependency = regexp.MustCompile(`^(?P<id>.+).external-dependency.werf.io$`)

var annotationKeyHumanLegacyExternalDependencyResource = "<name>.external-dependency.werf.io/resource"
var annotationKeyPatternLegacyExternalDependencyResource = regexp.MustCompile(`^(?P<id>.+).external-dependency.werf.io/resource$`)

var annotationKeyHumanLegacyExternalDependencyNamespace = "<name>.external-dependency.werf.io/namespace"
var annotationKeyPatternLegacyExternalDependencyNamespace = regexp.MustCompile(`^(?P<id>.+).external-dependency.werf.io/namespace$`)

func validateHook(res *unstructured.Unstructured) error {
	if key, value, found := FindAnnotationOrLabelByKeyPattern(res.GetAnnotations(), annotationKeyPatternHook); found {
		if value == "" {
			return fmt.Errorf("invalid value %q for annotation %q, expected non-empty string value", value, key)
		}

		for _, hookType := range strings.Split(value, ",") {
			hookType = strings.TrimSpace(hookType)
			if hookType == "" {
				return fmt.Errorf("invalid value %q for annotation %q, one of the comma-separated values is empty", value, key)
			}

			switch hookType {
			case string(release.HookPreInstall),
				string(release.HookPostInstall),
				string(release.HookPreUpgrade),
				string(release.HookPostUpgrade),
				string(release.HookPreRollback),
				string(release.HookPostRollback),
				string(release.HookPreDelete),
				string(release.HookPostDelete),
				string(release.HookTest),
				"test-success":
			default:
				return fmt.Errorf("value %q for annotation %q is not supported", value, key)
			}
		}
	} else {
		panic("hook resource must have hook annotation")
	}

	return nil
}

func validateWeight(unstruct *unstructured.Unstructured) error {
	if IsHook(unstruct.GetAnnotations()) {
		if key, value, found := FindAnnotationOrLabelByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternHookWeight); found {
			if value == "" {
				return fmt.Errorf("invalid value %q for annotation %q, expected non-empty integer value", value, key)
			}

			if _, err := strconv.Atoi(value); err != nil {
				return fmt.Errorf("invalid value %q for annotation %q, expected integer value", value, key)
			}
		}
	}

	if key, value, found := FindAnnotationOrLabelByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternWeight); found {
		if value == "" {
			return fmt.Errorf("invalid value %q for annotation %q, expected non-empty integer value", value, key)
		}

		if _, err := strconv.Atoi(value); err != nil {
			return fmt.Errorf("invalid value %q for annotation %q, expected integer value", value, key)
		}
	}

	return nil
}

func validateResourcePolicy(unstruct *unstructured.Unstructured) error {
	if key, value, found := FindAnnotationOrLabelByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternResourcePolicy); found {
		if value == "" {
			return fmt.Errorf("invalid value %q for annotation %q, expected non-empty string value", value, key)
		}

		switch value {
		case "keep":
		default:
			return fmt.Errorf("invalid unknown value %q for annotation %q", value, key)
		}
	}

	return nil
}

func validateDeletePolicy(unstruct *unstructured.Unstructured) error {
	annotations := unstruct.GetAnnotations()

	if IsHook(annotations) {
		if key, value, found := FindAnnotationOrLabelByKeyPattern(annotations, annotationKeyPatternHookDeletePolicy); found {
			if value == "" {
				return fmt.Errorf("invalid value %q for annotation %q, expected non-empty string value", value, key)
			}

			for _, hookDeletePolicy := range strings.Split(value, ",") {
				hookDeletePolicy = strings.TrimSpace(hookDeletePolicy)
				if hookDeletePolicy == "" {
					return fmt.Errorf("invalid value %q for annotation %q, one of the comma-separated values is empty", value, key)
				}

				switch hookDeletePolicy {
				case string(release.HookSucceeded),
					string(release.HookFailed),
					string(release.HookBeforeHookCreation):
				default:
					return fmt.Errorf("value %q for annotation %q is not supported", value, key, hookDeletePolicy)
				}
			}
		}
	}

	if key, value, found := FindAnnotationOrLabelByKeyPattern(annotations, annotationKeyPatternDeletePolicy); found {
		if value == "" {
			return fmt.Errorf("invalid value %q for annotation %q, expected non-empty string value", value, key)
		}

		for _, deletePolicy := range strings.Split(value, ",") {
			deletePolicy = strings.TrimSpace(deletePolicy)
			if deletePolicy == "" {
				return fmt.Errorf("invalid value %q for annotation %q, one of the comma-separated values is empty", value, key)
			}

			switch deletePolicy {
			case string(common.DeletePolicySucceeded),
				string(common.DeletePolicyFailed),
				string(common.DeletePolicyBeforeCreation):
			default:
				return fmt.Errorf("value %q for annotation %q is not supported", value, key, deletePolicy)
			}
		}
	}

	return nil
}

func validateReplicasOnCreation(unstruct *unstructured.Unstructured) error {
	if key, value, found := FindAnnotationOrLabelByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternReplicasOnCreation); found {
		if value == "" {
			return fmt.Errorf("invalid value %q for annotation %q, expected non-empty numeric value", value, key)
		}

		replicas, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid value %q for annotation %q, value must be a number", value, key)
		}

		if replicas < 0 {
			return fmt.Errorf("invalid value %q for annotation %q, value must be a positive number or zero", value, key)
		}
	}

	return nil
}

func validateTrack(unstruct *unstructured.Unstructured) error {
	if key, value, found := FindAnnotationOrLabelByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternFailMode); found {
		if value == "" {
			return fmt.Errorf("invalid value %q for annotation %q, expected non-empty string value", value, key)
		}

		switch value {
		case string(multitrack.IgnoreAndContinueDeployProcess):
		case string(multitrack.FailWholeDeployProcessImmediately):
		case string(multitrack.HopeUntilEndOfDeployProcess):
		default:
			return fmt.Errorf("invalid unknown value %q for annotation %q", value, key)
		}
	}

	if key, value, found := FindAnnotationOrLabelByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternFailuresAllowedPerReplica); found {
		if value == "" {
			return fmt.Errorf("invalid value %q for annotation %q, expected non-empty integer value", value, key)
		}

		if failures, err := strconv.Atoi(value); err != nil {
			return fmt.Errorf("invalid value %q for annotation %q, expected integer value", value, key)
		} else if failures < 0 {
			return fmt.Errorf("invalid value %q for annotation %q, expected non-negative integer value", value, key)
		}
	}

	if annotations, found := FindAnnotationsOrLabelsByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternIgnoreReadinessProbeFailsFor); found {
		for key, value := range annotations {
			keyMatches := annotationKeyPatternIgnoreReadinessProbeFailsFor.FindStringSubmatch(key)
			if keyMatches == nil {
				return fmt.Errorf("invalid key for annotation %q", key)
			}

			containerSubexpIndex := annotationKeyPatternIgnoreReadinessProbeFailsFor.SubexpIndex("container")
			if containerSubexpIndex == -1 {
				return fmt.Errorf("invalid regexp pattern %q for annotation %q", annotationKeyPatternIgnoreReadinessProbeFailsFor.String(), key)
			}

			if len(keyMatches) < containerSubexpIndex+1 {
				return fmt.Errorf("can't parse container name for annotation %q", key)
			}

			if value == "" {
				return fmt.Errorf("invalid value %q for annotation %q, expected non-empty value", value, key)
			}

			duration, err := time.ParseDuration(value)
			if err != nil {
				return fmt.Errorf("invalid value %q for annotation %q, expected valid duration", value, key)
			}

			if math.Signbit(duration.Seconds()) {
				return fmt.Errorf("invalid value %q for annotation %q, expected positive duration value", value, key)
			}
		}
	}

	if key, value, found := FindAnnotationOrLabelByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternLogRegex); found {
		if value == "" {
			return fmt.Errorf("invalid value %q for annotation %q, expected non-empty string value", value, key)
		}

		if _, err := regexp.Compile(value); err != nil {
			return fmt.Errorf("invalid value %q for annotation %q, expected valid regexp", value, key)
		}
	}

	if annotations, found := FindAnnotationsOrLabelsByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternLogRegexFor); found {
		for key, value := range annotations {
			keyMatches := annotationKeyPatternLogRegexFor.FindStringSubmatch(key)
			if keyMatches == nil {
				return fmt.Errorf("invalid key for annotation %q", key)
			}

			containerSubexpIndex := annotationKeyPatternLogRegexFor.SubexpIndex("container")
			if containerSubexpIndex == -1 {
				return fmt.Errorf("invalid regexp pattern %q for annotation %q", annotationKeyPatternLogRegexFor.String(), key)
			}

			if len(keyMatches) < containerSubexpIndex+1 {
				return fmt.Errorf("can't parse container name for annotation %q", key)
			}

			if value == "" {
				return fmt.Errorf("invalid value %q for annotation %q, expected non-empty value", value, key)
			}

			if _, err := regexp.Compile(value); err != nil {
				return fmt.Errorf("invalid value %q for annotation %q, expected valid regular expression", value, key)
			}
		}
	}

	if key, value, found := FindAnnotationOrLabelByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternNoActivityTimeout); found {
		if value == "" {
			return fmt.Errorf("invalid value %q for annotation %q, expected non-empty duration value", value, key)
		}

		duration, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("invalid value %q for annotation %q, expected valid duration", value, key)
		}

		if duration.Seconds() < 0 {
			return fmt.Errorf("invalid value %q for annotation %q, expected non-negative duration value", value, key)
		}
	}

	if key, value, found := FindAnnotationOrLabelByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternShowLogsOnlyForContainers); found {
		if value == "" {
			return fmt.Errorf("invalid value %q for annotation %q, expected non-empty string value", value, key)
		}

		if strings.Contains(value, ",") {
			for _, container := range strings.Split(value, ",") {
				container = strings.TrimSpace(container)
				if container == "" {

					return fmt.Errorf("invalid value %q for annotation %q, one of the comma-separated values is empty", value, key)
				}
			}
		}
	}

	if key, value, found := FindAnnotationOrLabelByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternShowServiceMessages); found {
		if value == "" {
			return fmt.Errorf("invalid value %q for annotation %q, expected non-empty boolean value", value, key)
		}

		if _, err := strconv.ParseBool(value); err != nil {
			return fmt.Errorf("invalid value %q for annotation %q, expected boolean value", value, key)
		}
	}

	if key, value, found := FindAnnotationOrLabelByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternSkipLogs); found {
		if value == "" {
			return fmt.Errorf("invalid value %q for annotation %q, expected non-empty boolean value", value, key)
		}

		if _, err := strconv.ParseBool(value); err != nil {
			return fmt.Errorf("invalid value %q for annotation %q, expected boolean value", value, key)
		}
	}

	if key, value, found := FindAnnotationOrLabelByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternSkipLogsForContainers); found {
		if value == "" {
			return fmt.Errorf("invalid value %q for annotation %q, expected non-empty string value", value, key)
		}

		if strings.Contains(value, ",") {
			for _, container := range strings.Split(value, ",") {
				container = strings.TrimSpace(container)
				if container == "" {
					return fmt.Errorf("invalid value %q for annotation %q, one of the comma-separated values is empty", value, key)
				}
			}
		}
	}

	if key, value, found := FindAnnotationOrLabelByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternTrackTerminationMode); found {
		if value == "" {
			return fmt.Errorf("invalid value %q for annotation %q, expected non-empty string value", value, key)
		}

		switch value {
		case string(multitrack.WaitUntilResourceReady):
		case string(multitrack.NonBlocking):
		default:
			return fmt.Errorf("invalid unknown value %q for annotation %q", value, key)
		}
	}

	return nil
}

func validateInternalDependencies(unstruct *unstructured.Unstructured) error {
	if annotations, found := FindAnnotationsOrLabelsByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternDependency); found {
		for key, value := range annotations {
			keyMatches := annotationKeyPatternDependency.FindStringSubmatch(key)
			if keyMatches == nil {
				return fmt.Errorf("invalid key for annotation %q", key)
			}

			idSubexpIndex := annotationKeyPatternDependency.SubexpIndex("id")
			if idSubexpIndex == -1 {
				return fmt.Errorf("invalid regexp pattern %q for annotation %q", annotationKeyPatternDependency.String(), key)
			}

			if len(keyMatches) < idSubexpIndex+1 {
				return fmt.Errorf("can't parse dependency id from annotation key %q", key)
			}

			if value != "" {
				valueElems := strings.Split(value, ":")

				if len(valueElems) != 3 && len(valueElems) != 4 {
					return fmt.Errorf(`invalid format of value %q for annotation %q, should be: apiVersion:kind[:namespace]:name or empty`, value, key)
				}
			}
		}
	}

	return nil
}

func validateExternalDependencies(unstruct *unstructured.Unstructured) error {
	if annotations, found := FindAnnotationsOrLabelsByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternExternalDependency); found {
		for key, value := range annotations {
			keyMatches := annotationKeyPatternExternalDependency.FindStringSubmatch(key)
			if keyMatches == nil {
				return fmt.Errorf("invalid key for annotation %q", key)
			}

			idSubexpIndex := annotationKeyPatternExternalDependency.SubexpIndex("id")
			if idSubexpIndex == -1 {
				return fmt.Errorf("invalid regexp pattern %q for annotation %q", annotationKeyPatternExternalDependency.String(), key)
			}

			if len(keyMatches) < idSubexpIndex+1 {
				return fmt.Errorf("can't parse external dependency id from annotation key %q", key)
			}

			valueElems := strings.Split(value, ":")

			if len(valueElems) != 3 && len(valueElems) != 4 {
				return fmt.Errorf(`invalid format of value %q for annotation %q, should be: apiVersion:kind[:namespace]:name`, value, key)
			}
		}
	}

	if annotations, found := FindAnnotationsOrLabelsByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternLegacyExternalDependencyResource); found {
		for key, value := range annotations {
			keyMatches := annotationKeyPatternLegacyExternalDependencyResource.FindStringSubmatch(key)
			if keyMatches == nil {
				return fmt.Errorf("invalid key for annotation %q", key)
			}

			idSubexpIndex := annotationKeyPatternLegacyExternalDependencyResource.SubexpIndex("id")
			if idSubexpIndex == -1 {
				return fmt.Errorf("invalid regexp pattern %q for annotation %q", annotationKeyPatternLegacyExternalDependencyResource.String(), key)
			}

			if len(keyMatches) < idSubexpIndex+1 {
				return fmt.Errorf("can't parse external dependency id from annotation key %q", key)
			}

			valueElems := strings.Split(value, "/")

			if len(valueElems) != 2 {
				return fmt.Errorf(`invalid format of value %q for annotation %q, should be: type/name`, value, key)
			}

			switch valueElems[0] {
			case "":
				return fmt.Errorf("value %q of annotation %q can't have empty resource type", value, key)
			case "all":
				return fmt.Errorf(`"all" resource type in value %q of annotation %q is not allowed`, value, key)
			}

			resourceTypeParts := strings.Split(valueElems[0], ".")
			for _, part := range resourceTypeParts {
				if part == "" {
					return fmt.Errorf("resource type in value %q of annotation %q should have dots (.) delimiting only non-empty resource.version.group", value, key)
				}
			}

			if valueElems[1] == "" {
				return fmt.Errorf("in value %q of annotation %q resource name can't be empty", value, key)
			}
		}
	}

	if annotations, found := FindAnnotationsOrLabelsByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternLegacyExternalDependencyNamespace); found {
		for key, value := range annotations {
			keyMatches := annotationKeyPatternLegacyExternalDependencyNamespace.FindStringSubmatch(key)
			if keyMatches == nil {
				return fmt.Errorf("invalid key for annotation %q", key)
			}

			idSubexpIndex := annotationKeyPatternLegacyExternalDependencyNamespace.SubexpIndex("id")
			if idSubexpIndex == -1 {
				return fmt.Errorf("invalid regexp pattern %q for annotation %q", annotationKeyPatternLegacyExternalDependencyNamespace.String(), key)
			}

			if len(keyMatches) < idSubexpIndex+1 {
				return fmt.Errorf("can't parse external dependency id from annotation key %q", key)
			}

			if value == "" {
				return fmt.Errorf("invalid value %q for annotation %q, value must not be empty", value, key)
			}
		}
	}

	return nil
}

func onPreInstall(unstruct *unstructured.Unstructured) bool {
	_, value := lo.Must2(FindAnnotationOrLabelByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternHook))
	return value == string(release.HookPreInstall)
}

func onPostInstall(unstruct *unstructured.Unstructured) bool {
	_, value := lo.Must2(FindAnnotationOrLabelByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternHook))
	return value == string(release.HookPostInstall)
}

func onPreUpgrade(unstruct *unstructured.Unstructured) bool {
	_, value := lo.Must2(FindAnnotationOrLabelByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternHook))
	return value == string(release.HookPreUpgrade)
}

func onPostUpgrade(unstruct *unstructured.Unstructured) bool {
	_, value := lo.Must2(FindAnnotationOrLabelByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternHook))
	return value == string(release.HookPostUpgrade)
}

func onPreRollback(unstruct *unstructured.Unstructured) bool {
	_, value := lo.Must2(FindAnnotationOrLabelByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternHook))
	return value == string(release.HookPreRollback)
}

func onPostRollback(unstruct *unstructured.Unstructured) bool {
	_, value := lo.Must2(FindAnnotationOrLabelByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternHook))
	return value == string(release.HookPostRollback)
}

func onPreDelete(unstruct *unstructured.Unstructured) bool {
	_, value := lo.Must2(FindAnnotationOrLabelByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternHook))
	return value == string(release.HookPreDelete)
}

func onPostDelete(unstruct *unstructured.Unstructured) bool {
	_, value := lo.Must2(FindAnnotationOrLabelByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternHook))
	return value == string(release.HookPostDelete)
}

func onTest(unstruct *unstructured.Unstructured) bool {
	_, value := lo.Must2(FindAnnotationOrLabelByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternHook))
	return value == string(release.HookTest) || value == "test-success"
}

func onPreAnything(unstruct *unstructured.Unstructured) bool {
	return onPreInstall(unstruct) || onPreUpgrade(unstruct) || onPreRollback(unstruct) || onPreDelete(unstruct)
}

func onPostAnything(unstruct *unstructured.Unstructured) bool {
	return onPostInstall(unstruct) || onPostUpgrade(unstruct) || onPostRollback(unstruct) || onPostDelete(unstruct)
}

func keepOnDelete(unstruct *unstructured.Unstructured) bool {
	_, value, found := FindAnnotationOrLabelByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternResourcePolicy)
	if !found {
		return false
	}

	return value == "keep"
}

func recreate(unstruct *unstructured.Unstructured) bool {
	deletePolicies := deletePolicies(unstruct.GetAnnotations())

	return lo.Contains(deletePolicies, common.DeletePolicyBeforeCreation)
}

func defaultReplicasOnCreation(unstruct *unstructured.Unstructured) (replicas int, set bool) {
	if IsCRDFromGK(unstruct.GroupVersionKind().GroupKind()) {
		return 0, false
	}

	_, value, found := FindAnnotationOrLabelByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternReplicasOnCreation)
	if !found {
		return 0, false
	}

	replicas = lo.Must(strconv.Atoi(value))

	return replicas, true
}

func failMode(unstruct *unstructured.Unstructured) multitrack.FailMode {
	_, value, found := FindAnnotationOrLabelByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternFailMode)
	if !found {
		return multitrack.FailWholeDeployProcessImmediately
	}

	return multitrack.FailMode(value)
}

func failuresAllowed(unstruct *unstructured.Unstructured) int {
	_, value, found := FindAnnotationOrLabelByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternFailuresAllowedPerReplica)
	var failuresAllowed int
	if found {
		failuresAllowed = lo.Must(strconv.Atoi(value))
	} else {
		failuresAllowed = 1
	}

	replicas, replicasFound := lo.Must2(unstructured.NestedInt64(unstruct.UnstructuredContent(), "spec", "replicas"))

	if replicasFound {
		return int(replicas) * failuresAllowed
	} else {
		return failuresAllowed
	}
}

func ignoreReadinessProbeFailsForContainers(unstruct *unstructured.Unstructured) (durationByContainer map[string]time.Duration, set bool) {
	annotations, found := FindAnnotationsOrLabelsByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternIgnoreReadinessProbeFailsFor)
	if !found {
		return nil, false
	}

	for key, value := range annotations {
		keyMatches := annotationKeyPatternIgnoreReadinessProbeFailsFor.FindStringSubmatch(key)
		containerSubexpIndex := annotationKeyPatternIgnoreReadinessProbeFailsFor.SubexpIndex("container")
		container := keyMatches[containerSubexpIndex]
		duration := lo.Must(time.ParseDuration(value))

		durationByContainer[container] = duration
	}

	return durationByContainer, true
}

func logRegex(unstruct *unstructured.Unstructured) (regex *regexp.Regexp, set bool) {
	_, value, found := FindAnnotationOrLabelByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternLogRegex)
	if !found {
		return nil, false
	}

	return regexp.MustCompile(value), true
}

func logRegexesForContainers(unstruct *unstructured.Unstructured) (regexByContainer map[string]*regexp.Regexp, set bool) {
	annotations, found := FindAnnotationsOrLabelsByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternLogRegexFor)
	if !found {
		return nil, false
	}

	for key, value := range annotations {
		keyMatches := annotationKeyPatternLogRegexFor.FindStringSubmatch(key)
		containerSubexpIndex := annotationKeyPatternLogRegexFor.SubexpIndex("container")
		container := keyMatches[containerSubexpIndex]
		regexByContainer[container] = regexp.MustCompile(value)
	}

	return regexByContainer, true
}

func noActivityTimeout(unstruct *unstructured.Unstructured) (timeout *time.Duration, set bool) {
	_, value, found := FindAnnotationOrLabelByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternNoActivityTimeout)
	if !found {
		return nil, true
	}

	t := lo.Must(time.ParseDuration(value))

	return &t, true
}

func showLogsOnlyForContainers(unstruct *unstructured.Unstructured) (containers []string, set bool) {
	_, value, found := FindAnnotationOrLabelByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternShowLogsOnlyForContainers)
	if !found {
		return nil, false
	}

	for _, container := range strings.Split(value, ",") {
		containers = append(containers, strings.TrimSpace(container))
	}

	return containers, true
}

func showServiceMessages(unstruct *unstructured.Unstructured) bool {
	_, value, found := FindAnnotationOrLabelByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternShowServiceMessages)
	if !found {
		return false
	}

	showServiceMessages := lo.Must(strconv.ParseBool(value))

	return showServiceMessages
}

func skipLogs(unstruct *unstructured.Unstructured) bool {
	_, value, found := FindAnnotationOrLabelByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternSkipLogs)
	if !found {
		return false
	}

	skipLogs := lo.Must(strconv.ParseBool(value))

	return skipLogs
}

func skipLogsForContainers(unstruct *unstructured.Unstructured) (containers []string, set bool) {
	_, value, found := FindAnnotationOrLabelByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternSkipLogsForContainers)
	if !found {
		return nil, false
	}

	for _, container := range strings.Split(value, ",") {
		containers = append(containers, strings.TrimSpace(container))
	}

	return containers, true
}

func trackTerminationMode(unstruct *unstructured.Unstructured) multitrack.TrackTerminationMode {
	_, value, found := FindAnnotationOrLabelByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternTrackTerminationMode)
	if !found {
		return multitrack.WaitUntilResourceReady
	}

	return multitrack.TrackTerminationMode(value)
}

func deleteOnSucceeded(unstruct *unstructured.Unstructured) bool {
	deletePolicies := deletePolicies(unstruct.GetAnnotations())

	return lo.Contains(deletePolicies, common.DeletePolicySucceeded)
}

func deleteOnFailed(unstruct *unstructured.Unstructured) bool {
	deletePolicies := deletePolicies(unstruct.GetAnnotations())

	return lo.Contains(deletePolicies, common.DeletePolicyFailed)
}

func adoptableBy(unstruct *unstructured.Unstructured, releaseName, releaseNamespace string) (adoptable bool, nonAdoptableReason string) {
	nonAdoptableReasons := []string{}

	if key, value, found := FindAnnotationOrLabelByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternReleaseName); found {
		if value != releaseName {
			nonAdoptableReasons = append(nonAdoptableReasons, fmt.Sprintf(`annotation "%s=%s" must have value %q`, key, value, releaseName))
		}
	} else {
		nonAdoptableReasons = append(nonAdoptableReasons, fmt.Sprintf(`annotation %q not found, must be set to %q`, annotationKeyHumanReleaseName, value))
	}

	if key, value, found := FindAnnotationOrLabelByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternReleaseNamespace); found {
		if value != releaseNamespace {
			nonAdoptableReasons = append(nonAdoptableReasons, fmt.Sprintf(`annotation "%s=%s" must have value %q`, key, value, releaseNamespace))
		}
	} else {
		nonAdoptableReasons = append(nonAdoptableReasons, fmt.Sprintf(`annotation %q not found, must be set to %q`, annotationKeyHumanReleaseNamespace, value))
	}

	nonAdoptableReason = strings.Join(nonAdoptableReasons, ", ")

	return len(nonAdoptableReasons) == 0, nonAdoptableReason
}

func fixManagedFields(unstruct *unstructured.Unstructured) (changed bool, err error) {
	managedFields := unstruct.GetManagedFields()
	if len(managedFields) == 0 {
		return false, nil
	}

	var oursEntry metav1.ManagedFieldsEntry
	if e, found := lo.Find(managedFields, func(e metav1.ManagedFieldsEntry) bool {
		return e.Manager == common.DefaultFieldManager && e.Operation == metav1.ManagedFieldsOperationApply
	}); found {
		oursEntry = e
	} else {
		oursEntry = metav1.ManagedFieldsEntry{
			Manager:    common.DefaultFieldManager,
			Operation:  metav1.ManagedFieldsOperationApply,
			APIVersion: unstruct.GetAPIVersion(),
			Time:       lo.ToPtr(metav1.Now()),
			FieldsType: "FieldsV1",
			FieldsV1:   &metav1.FieldsV1{Raw: []byte("{}")},
		}
	}

	var fixedManagedFields []metav1.ManagedFieldsEntry

	fixedManagedFields = append(fixedManagedFields, differentSubresourceManagers(managedFields, oursEntry)...)

	if newManagedFields, newOursEntry, chngd := removeUndesirableManagers(managedFields, oursEntry); chngd {
		fixedManagedFields = append(fixedManagedFields, newManagedFields...)
		oursEntry = newOursEntry
		changed = true
	}

	if newManagedFields, chngd := exclusiveOwnershipForOurManager(managedFields, oursEntry); chngd {
		fixedManagedFields = append(fixedManagedFields, newManagedFields...)
		changed = true
	}

	if string(oursEntry.FieldsV1.Raw) != "{}" {
		fixedManagedFields = append(fixedManagedFields, oursEntry)
	}

	if changed {
		unstruct.SetManagedFields(fixedManagedFields)
	}

	return changed, nil
}

func differentSubresourceManagers(managedFields []metav1.ManagedFieldsEntry, oursEntry metav1.ManagedFieldsEntry) (newManagedFields []metav1.ManagedFieldsEntry) {
	for _, managedField := range managedFields {
		if managedField.Subresource != oursEntry.Subresource {
			newManagedFields = append(newManagedFields, managedField)
			continue
		}
	}

	return newManagedFields
}

func removeUndesirableManagers(managedFields []metav1.ManagedFieldsEntry, oursEntry metav1.ManagedFieldsEntry) (newManagedFields []metav1.ManagedFieldsEntry, newOursEntry metav1.ManagedFieldsEntry, changed bool) {
	oursFieldsByte := lo.Must(json.Marshal(oursEntry.FieldsV1))

	newOursEntry = oursEntry
	for _, managedField := range managedFields {
		if managedField.Subresource != oursEntry.Subresource {
			continue
		}

		fieldsByte := lo.Must(json.Marshal(managedField.FieldsV1))

		switch managedField.Manager {
		case common.OldFieldManager, common.KubectlEditFieldManager:
			merged, mergeChanged := lo.Must2(utls.MergeJson(fieldsByte, oursFieldsByte))
			if mergeChanged {
				oursFieldsByte = merged
				lo.Must0(newOursEntry.FieldsV1.UnmarshalJSON(merged))
			}

			changed = true
		case common.DefaultFieldManager:
			if managedField.Operation == metav1.ManagedFieldsOperationApply {
				continue
			}

			merged, mergeChanged := lo.Must2(utls.MergeJson(fieldsByte, oursFieldsByte))
			if mergeChanged {
				oursFieldsByte = merged
				lo.Must0(newOursEntry.FieldsV1.UnmarshalJSON(merged))
			}

			changed = true
		}
	}

	return newManagedFields, newOursEntry, changed
}

func exclusiveOwnershipForOurManager(managedFields []metav1.ManagedFieldsEntry, oursEntry metav1.ManagedFieldsEntry) (newManagedFields []metav1.ManagedFieldsEntry, changed bool) {
	oursFieldsByte := lo.Must(json.Marshal(oursEntry.FieldsV1))

	for _, managedField := range managedFields {
		if managedField.Subresource != oursEntry.Subresource {
			continue
		}

		fieldsByte := lo.Must(json.Marshal(managedField.FieldsV1))

		switch managedField.Manager {
		case common.OldFieldManager, common.KubectlEditFieldManager, common.DefaultFieldManager:
			continue
		default:
			subtracted, subtractChanged := lo.Must2(utls.SubtractJson(fieldsByte, oursFieldsByte))
			if !subtractChanged {
				newManagedFields = append(newManagedFields, managedField)
				continue
			}

			if string(subtracted) != "{}" {
				lo.Must0(managedField.FieldsV1.UnmarshalJSON(subtracted))
				newManagedFields = append(newManagedFields, managedField)
			}

			changed = true
		}
	}

	return newManagedFields, changed
}

func weight(unstruct *unstructured.Unstructured) int {
	var weightValue string
	if IsHook(unstruct.GetAnnotations()) {
		_, hookWeightValue, hookWeightFound := FindAnnotationOrLabelByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternHookWeight)

		_, generalWeightValue, weightFound := FindAnnotationOrLabelByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternWeight)

		if !hookWeightFound && !weightFound {
			return 0
		} else if weightFound {
			weightValue = generalWeightValue
		} else {
			weightValue = hookWeightValue
		}
	} else {
		var found bool
		_, weightValue, found = FindAnnotationOrLabelByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternWeight)
		if !found {
			return 0
		}
	}

	weight := lo.Must(strconv.Atoi(weightValue))

	return weight
}

func deletePolicies(annotations map[string]string) []common.DeletePolicy {
	var deletePolicies []common.DeletePolicy
	if IsHook(annotations) {
		hookDeletePoliciesValues, hookDeletePoliciesFound := FindAnnotationsOrLabelsByKeyPattern(annotations, annotationKeyPatternHookDeletePolicy)

		generalDeletePoliciesValues, generalDeletePoliciesFound := FindAnnotationsOrLabelsByKeyPattern(annotations, annotationKeyPatternDeletePolicy)

		if !hookDeletePoliciesFound && !generalDeletePoliciesFound {
			deletePolicies = append(deletePolicies, common.DeletePolicyBeforeCreation)
		} else if generalDeletePoliciesFound {
			for _, generalDeletePolicyValue := range generalDeletePoliciesValues {
				deletePolicies = append(deletePolicies, common.DeletePolicy(generalDeletePolicyValue))
			}
		} else {
			for _, hookDeletePolicyValue := range hookDeletePoliciesValues {
				switch hookDeletePolicyValue {
				case string(release.HookSucceeded):
					deletePolicies = append(deletePolicies, common.DeletePolicySucceeded)
				case string(release.HookFailed):
					deletePolicies = append(deletePolicies, common.DeletePolicyFailed)
				case string(release.HookBeforeHookCreation):
					deletePolicies = append(deletePolicies, common.DeletePolicyBeforeCreation)
				}
			}
		}
	} else {
		deletePoliciesValues, deletePoliciesFound := FindAnnotationsOrLabelsByKeyPattern(annotations, annotationKeyPatternDeletePolicy)
		if deletePoliciesFound {
			for _, deletePolicyValue := range deletePoliciesValues {
				deletePolicies = append(deletePolicies, common.DeletePolicy(deletePolicyValue))
			}
		}
	}

	return deletePolicies
}

func manualInternalDependencies(unstruct *unstructured.Unstructured, defaultNamespace string) (dependencies []*depnd.InternalDependency, set bool) {
	deps := map[string]*depnd.InternalDependency{}
	if annotations, found := FindAnnotationsOrLabelsByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternDependency); found {
		for key, value := range annotations {
			matches := annotationKeyPatternDependency.FindStringSubmatch(key)
			idSubexpIndex := annotationKeyPatternDependency.SubexpIndex("id")
			depID := matches[idSubexpIndex]
			valParts := strings.Split(value, ":")
			depApiVersionParts := strings.SplitN(valParts[0], "/", 2)

			var gvk schema.GroupVersionKind
			if len(depApiVersionParts) == 1 {
				gvk = schema.GroupVersionKind{
					Version: depApiVersionParts[0],
					Kind:    valParts[1],
				}
			} else {
				gvk = schema.GroupVersionKind{
					Group:   depApiVersionParts[0],
					Version: depApiVersionParts[1],
					Kind:    valParts[1],
				}
			}

			var depNamespace string
			if len(valParts) == 4 {
				depNamespace = valParts[2]
			}

			depName := valParts[len(valParts)-1]

			dep := depnd.NewInternalDependency(
				[]string{depName},
				[]string{depNamespace},
				[]string{gvk.Group},
				[]string{gvk.Version},
				[]string{gvk.Kind},
				depnd.InternalDependencyOptions{
					DefaultNamespace: defaultNamespace,
				},
			)
			deps[depID] = dep
		}
	}

	return lo.Values(deps), len(deps) > 0
}

func autoInternalDependencies(unstruct *unstructured.Unstructured, defaultNamespace string) (dependencies []*depnd.InternalDependency, set bool) {
	depDetector := depnddetctr.NewInternalDependencyDetector(depnddetctr.InternalDependencyDetectorOptions{
		DefaultNamespace: defaultNamespace,
	})
	dependencies = depDetector.Detect(unstruct)

	return dependencies, len(dependencies) > 0
}

func externalDependencies(unstruct *unstructured.Unstructured, defaultNamespace string, mapper meta.ResettableRESTMapper, discoveryClient discovery.CachedDiscoveryInterface) (dependencies []*depnd.ExternalDependency, set bool, err error) {
	deps := externalDeps(unstruct, defaultNamespace, mapper)

	legacyExtDeps := map[string]*depnd.ExternalDependency{}
	// Pretend that we don't have any external dependencies when we don't have cluster access, since we need cluster access to map GVR to GVK.
	if mapper != nil && discoveryClient != nil {
		var err error
		legacyExtDeps, err = legacyExternalDeps(unstruct, defaultNamespace, mapper, discoveryClient)
		if err != nil {
			return nil, false, fmt.Errorf("error getting legacy external dependencies: %w", err)
		}
	}

	duplResult := lo.Values(lo.Assign(legacyExtDeps, deps))
	uniqResult := lo.UniqBy(duplResult, func(d *depnd.ExternalDependency) string {
		return d.ID()
	})

	return uniqResult, len(uniqResult) > 0, nil
}

func externalDeps(unstruct *unstructured.Unstructured, defaultNamespace string, mapper meta.ResettableRESTMapper) map[string]*depnd.ExternalDependency {
	deps := map[string]*depnd.ExternalDependency{}
	if annotations, found := FindAnnotationsOrLabelsByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternExternalDependency); found {
		for key, value := range annotations {
			matches := annotationKeyPatternExternalDependency.FindStringSubmatch(key)
			idSubexpIndex := annotationKeyPatternExternalDependency.SubexpIndex("id")
			depID := matches[idSubexpIndex]
			valParts := strings.Split(value, ":")
			depApiVersionParts := strings.SplitN(valParts[0], "/", 2)

			var gvk schema.GroupVersionKind
			if len(depApiVersionParts) == 1 {
				gvk = schema.GroupVersionKind{
					Version: depApiVersionParts[0],
					Kind:    valParts[1],
				}
			} else {
				gvk = schema.GroupVersionKind{
					Group:   depApiVersionParts[0],
					Version: depApiVersionParts[1],
					Kind:    valParts[1],
				}
			}

			var depNamespace string
			if len(valParts) == 4 {
				depNamespace = valParts[2]
			}

			depName := valParts[len(valParts)-1]

			dep := depnd.NewExternalDependency(
				depName,
				depNamespace,
				gvk,
				depnd.ExternalDependencyOptions{
					DefaultNamespace: defaultNamespace,
					Mapper:           mapper,
				},
			)

			deps[depID] = dep
		}
	}

	return deps
}

func legacyExternalDeps(unstruct *unstructured.Unstructured, defaultNamespace string, mapper meta.ResettableRESTMapper, discoveryClient discovery.CachedDiscoveryInterface) (map[string]*depnd.ExternalDependency, error) {
	deps := map[string]*depnd.ExternalDependency{}

	type DepInfo struct {
		Name      string
		Namespace string
		Type      string
	}
	extDepInfos := map[string]*DepInfo{}

	if annotations, found := FindAnnotationsOrLabelsByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternLegacyExternalDependencyResource); found {
		for key, value := range annotations {
			matches := annotationKeyPatternLegacyExternalDependencyResource.FindStringSubmatch(key)
			idSubexpIndex := annotationKeyPatternLegacyExternalDependencyResource.SubexpIndex("id")
			extDepID := matches[idSubexpIndex]
			extDepType := strings.Split(value, "/")[0]
			extDepName := strings.Split(value, "/")[1]

			extDepInfos[extDepID] = &DepInfo{
				Name: extDepName,
				Type: extDepType,
			}
		}
	}

	if annotations, found := FindAnnotationsOrLabelsByKeyPattern(unstruct.GetAnnotations(), annotationKeyPatternLegacyExternalDependencyNamespace); found {
		for key, value := range annotations {
			matches := annotationKeyPatternLegacyExternalDependencyNamespace.FindStringSubmatch(key)
			idSubexpIndex := annotationKeyPatternLegacyExternalDependencyNamespace.SubexpIndex("id")
			extDepID := matches[idSubexpIndex]
			extDepNamespace := value

			if extDepInfo, hasKey := extDepInfos[extDepID]; hasKey {
				extDepInfo.Namespace = extDepNamespace
			}
		}
	}

	for extDepID, extDepInfo := range extDepInfos {
		gvk, err := utls.ParseKubectlResourceStringtoGVK(extDepInfo.Type, mapper, discoveryClient)
		if err != nil {
			return nil, fmt.Errorf("error parsing external dependency resource type %q for dependency %q (namespace: %q): %w", extDepInfo.Type, extDepInfo.Name, extDepInfo.Namespace, err)
		}

		dep := depnd.NewExternalDependency(
			extDepInfo.Name,
			extDepInfo.Namespace,
			gvk,
			depnd.ExternalDependencyOptions{
				DefaultNamespace: defaultNamespace,
				Mapper:           mapper,
			},
		)
		deps[extDepID] = dep
	}

	return deps, nil
}
