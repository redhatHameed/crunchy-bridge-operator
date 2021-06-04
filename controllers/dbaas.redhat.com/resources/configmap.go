/*
 * Copyright (C) 2020 Red Hat, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package resources

import (
	dbaasredhatcomv1alpha1 "github.com/CrunchyData/crunchy-bridge-operator/apis/dbaas.redhat.com/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var labels = map[string]string{"related-to": "dbaas-operator", "type": "dbaas-provider-registration"}

func OwnedBridgeRegistrationConfigMap(inventory *dbaasredhatcomv1alpha1.CrunchyBridgeInventory) (c *corev1.ConfigMap) {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "crunchy-bridge-provider-registration",
			Namespace: "crunchy-bridge-operator-system",
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(inventory, schema.GroupVersionKind{
					Group:   dbaasredhatcomv1alpha1.GroupVersion.Group,
					Version: dbaasredhatcomv1alpha1.GroupVersion.Version,
					Kind:    inventory.Kind,
				}),
			},
		},
	}
}
func BridgeRegistrationConfigMapData() map[string]string {
	credentialsFields := `
publicApiKey.name: publicApiKey
publicApiKey.masked: false
privateApiSecret.name: privateApiSecret
privateApiSecret.masked: true
`
	data := map[string]string{
		"provider":           "Crunchy Bridge Postgres",
		"inventory_kind":     "CrunchyBridgeInventory",
		"connection_kind":    "CrunchyBridgeConnection",
		"credentials_fields": credentialsFields,
	}
	return data
}
func BridgeRegistrationConfigMap(inventory *dbaasredhatcomv1alpha1.CrunchyBridgeInventory) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "crunchy-bridge-provider-registration",
			Namespace: "crunchy-bridge-operator-system",
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(inventory, schema.GroupVersionKind{
					Group:   dbaasredhatcomv1alpha1.GroupVersion.Group,
					Version: dbaasredhatcomv1alpha1.GroupVersion.Version,
					Kind:    inventory.Kind,
				}),
			},
		},
	}
}
