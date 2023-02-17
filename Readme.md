# Mutate-Me

This is a mutation admission controller for kubernetes. 

## The Dev environment

Use this to build a [kind](https://kind.sigs.k8s.io/docs/user/quick-start/) environment for your local testing.

```
➜  mutate-me git:(master) ✗ kind create cluster --name dev-cluster --image kindest/node:v1.23.13

Creating cluster "dev-cluster" ...
 ✓ Ensuring node image (kindest/node:v1.23.13) 🖼 
 ✓ Preparing nodes 📦  
 ✓ Writing configuration 📜 
 ✓ Starting control-plane 🕹️ 
 ✓ Installing CNI 🔌 
 ✓ Installing StorageClass 💾 
Set kubectl context to "kind-dev-cluster"
You can now use your cluster with:

kubectl cluster-info --context kind-dev-cluster

Have a nice day! 👋
