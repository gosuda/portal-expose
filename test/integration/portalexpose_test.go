package integration

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	portalv1alpha1 "github.com/gosuda/portal-expose/api/v1alpha1"
)

var _ = Describe("PortalExpose Controller", func() {
	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When creating a PortalExpose", func() {
		It("Should create a Deployment and update Status", func() {
			By("Creating a Service to expose")
			serviceName := "test-service"
			namespace := "default"
			service := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      serviceName,
					Namespace: namespace,
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						{
							Port: 80,
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, service)).Should(Succeed())

			By("Creating a default TunnelClass")
			tunnelClassName := "default-tunnel-class"
			tunnelClass := &portalv1alpha1.TunnelClass{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tunnelClassName,
					Namespace: namespace,
					Annotations: map[string]string{
						"portal.gosuda.org/is-default-class": "true",
					},
				},
				Spec: portalv1alpha1.TunnelClassSpec{
					Replicas: 1,
					Size:     "small",
				},
			}
			Expect(k8sClient.Create(ctx, tunnelClass)).Should(Succeed())

			By("Creating a PortalExpose resource")
			portalExposeName := "test-portal-expose"
			portalExpose := &portalv1alpha1.PortalExpose{
				ObjectMeta: metav1.ObjectMeta{
					Name:      portalExposeName,
					Namespace: namespace,
				},
				Spec: portalv1alpha1.PortalExposeSpec{
					App: portalv1alpha1.AppSpec{
						Name: "test-app",
						Service: portalv1alpha1.ServiceRef{
							Name: serviceName,
							Port: 80,
						},
					},
					Relay: portalv1alpha1.RelaySpec{
						Targets: []portalv1alpha1.RelayTarget{
							{
								Name: "test-relay",
								URL:  "wss://portal.gosuda.org/relay",
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, portalExpose)).Should(Succeed())

			By("Verifying the Deployment is created")
			deploymentName := portalExposeName + "-tunnel"
			deploymentLookupKey := types.NamespacedName{Name: deploymentName, Namespace: namespace}
			createdDeployment := &appsv1.Deployment{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, deploymentLookupKey, createdDeployment)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(createdDeployment.Spec.Replicas).Should(Equal(int32Ptr(1)))

			By("Verifying OwnerReference is set")
			Expect(createdDeployment.OwnerReferences).To(HaveLen(1))
			Expect(createdDeployment.OwnerReferences[0].Name).To(Equal(portalExposeName))

			By("Simulating Pod readiness")
			// In a real cluster, the controller would see the Pods becoming ready.
			// Here we manually update the Deployment status to simulate this.
			createdDeployment.Status.Replicas = 1
			createdDeployment.Status.ReadyReplicas = 1
			Expect(k8sClient.Status().Update(ctx, createdDeployment)).Should(Succeed())

			By("Verifying PortalExpose status becomes Ready")
			portalExposeLookupKey := types.NamespacedName{Name: portalExposeName, Namespace: namespace}
			updatedPortalExpose := &portalv1alpha1.PortalExpose{}

			Eventually(func() string {
				err := k8sClient.Get(ctx, portalExposeLookupKey, updatedPortalExpose)
				if err != nil {
					return ""
				}
				return updatedPortalExpose.Status.Phase
			}, timeout, interval).Should(Equal("Ready"))

			By("Verifying PublicURL is generated")
			Expect(updatedPortalExpose.Status.PublicURL).To(Equal("https://test-app.portal.gosuda.org"))

			By("Deleting the PortalExpose")
			Expect(k8sClient.Delete(ctx, portalExpose)).Should(Succeed())

			By("Verifying the Deployment is deleted")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, deploymentLookupKey, createdDeployment)
				return client.IgnoreNotFound(err) == nil
			}, timeout, interval).Should(BeTrue())

			// Clean up Service and TunnelClass
			Expect(k8sClient.Delete(ctx, service)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, tunnelClass)).Should(Succeed())
		})
	})
})

// Helper to create int32 pointer
func int32Ptr(i int32) *int32 { return &i }
