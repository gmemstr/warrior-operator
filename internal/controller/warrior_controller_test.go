/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8sresource "k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	warriorv1alpha1 "git.gmem.ca/arch/warrior-operator/api/v1alpha1"
)

var _ = Describe("Warrior Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		typeNamespacedNameWarrior := types.NamespacedName{
			Name:      fmt.Sprintf("%s-warrior", resourceName),
			Namespace: "default",
		}
		warrior := &warriorv1alpha1.Warrior{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind Warrior")
			_ = os.Setenv("WARRIOR_IMAGE_BASE", "atdr.meo.ws/archiveteam")
			err := k8sClient.Get(ctx, typeNamespacedName, warrior)
			if err != nil && errors.IsNotFound(err) {
				resource := &warriorv1alpha1.Warrior{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: warriorv1alpha1.WarriorSpec{
						Project: "cohost",
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &warriorv1alpha1.Warrior{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance Warrior")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &WarriorReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
		})
		It("should successfully change minimum scale", func() {
			By("Reconciling the created resource")
			controllerReconciler := &WarriorReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			deployment := &appsv1.Deployment{}
			err = k8sClient.Get(ctx, typeNamespacedNameWarrior, deployment)
			Expect(err).NotTo(HaveOccurred())
			Expect(*deployment.Spec.Replicas).To(Equal(int32(0)))

			resource := &warriorv1alpha1.Warrior{}
			err = k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())
			resource.Spec.Scaling.Minimum = 1
			err = k8sClient.Update(ctx, resource)
			Expect(err).NotTo(HaveOccurred())
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			err = k8sClient.Get(ctx, typeNamespacedNameWarrior, deployment)
			Expect(err).NotTo(HaveOccurred())
			Expect(*deployment.Spec.Replicas).To(Equal(int32(1)))
		})

		It("should successfully change downloader and concurrency", func() {
			By("Reconciling the created resource")
			controllerReconciler := &WarriorReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			deployment := &appsv1.Deployment{}
			err = k8sClient.Get(ctx, typeNamespacedNameWarrior, deployment)
			Expect(err).NotTo(HaveOccurred())
			for _, v := range deployment.Spec.Template.Spec.Containers[0].Env {
				if v.Name == VAR_DOWNLOADER {
					Expect(v.Value).To(Equal(""))
				}
				if v.Name == VAR_CONCURRENT_ITEMS {
					Expect(v.Value).To(Equal("0"))
				}
			}

			resource := &warriorv1alpha1.Warrior{}
			err = k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())
			resource.Spec.Downloader = "test"
			resource.Spec.Scaling.Concurrency = 5
			err = k8sClient.Update(ctx, resource)
			Expect(err).NotTo(HaveOccurred())
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			err = k8sClient.Get(ctx, typeNamespacedNameWarrior, deployment)
			Expect(err).NotTo(HaveOccurred())
			for _, v := range deployment.Spec.Template.Spec.Containers[0].Env {
				if v.Name == VAR_DOWNLOADER {
					Expect(v.Value).To(Equal("test"))
				}
				if v.Name == VAR_CONCURRENT_ITEMS {
					Expect(v.Value).To(Equal("5"))
				}
			}
		})

		It("should successfully change cache size", func() {
			By("Reconciling the created resource")
			controllerReconciler := &WarriorReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			defaultSize, err := k8sresource.ParseQuantity("500Mi")
			Expect(err).NotTo(HaveOccurred())

			deployment := &appsv1.Deployment{}
			err = k8sClient.Get(ctx, typeNamespacedNameWarrior, deployment)
			Expect(err).NotTo(HaveOccurred())
			Expect(*deployment.Spec.Template.Spec.Volumes[0].EmptyDir.SizeLimit).To(Equal(defaultSize))

			newSize, err := k8sresource.ParseQuantity("10Mi")
			Expect(err).NotTo(HaveOccurred())
			resource := &warriorv1alpha1.Warrior{}
			err = k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())
			resource.Spec.Resources.CacheSize = newSize.String()
			err = k8sClient.Update(ctx, resource)
			Expect(err).NotTo(HaveOccurred())
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			err = k8sClient.Get(ctx, typeNamespacedNameWarrior, deployment)
			Expect(err).NotTo(HaveOccurred())
			Expect(*deployment.Spec.Template.Spec.Volumes[0].EmptyDir.SizeLimit).To(Equal(newSize))
		})
	})
})
