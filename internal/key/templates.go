package key

const AppCRTemplate = `
{{- .UserConfigConfigMap -}}
---
{{ .UserConfigSecret -}}
---
{{ .AppCR -}}
`

const CatalogCRTemplate = `
{{- .ConfigMap -}}
---
{{ .Secret -}}
---
{{ .CatalogCR -}}
`

const ClusterAWSCRsTemplate = `
{{- .ClusterCR -}}
---
{{ .AWSClusterCR -}}
---
{{ .G8sControlPlaneCR -}}
---
{{ .AWSControlPlaneCR -}}
`

const ClusterAzureCRsTemplate = `
{{- .ProviderClusterCR -}}
---
{{ .ClusterCR -}}
---
{{ .MasterMachineCR -}}
`

const ClusterCAPACRsTemplate = `
{{- .ClusterCR -}}
---
{{ .AWSClusterCR -}}
---
{{ .KubeadmControlPlaneCR -}}
---
{{ .AWSMachineTemplateCR -}}
---
{{ .AWSClusterRoleIdentityCR -}}
---
{{ .BastionBootstrapSecret -}}
---
{{ .BastionMachineDeploymentCR -}}
---
{{ .BastionAWSMachineTemplateCR -}}
`

const MachineDeploymentCRsTemplate = `
{{- .MachineDeploymentCR -}}
---
{{ .AWSMachineDeploymentCR -}}
`

const NetworkPoolCRsTemplate = `
{{- .NetworkPoolCR -}}
`

const MachinePoolAWSCRsTemplate = `
{{- .ProviderMachinePoolCR -}}
---
{{ .MachinePoolCR -}}
---
{{ .KubeadmConfigCR -}}
`

const MachinePoolAzureCRsTemplate = `
{{- .ProviderMachinePoolCR -}}
---
{{ .MachinePoolCR -}}
---
{{ .SparkCR -}}
`

const BastionIgnitionTemplate = `{
   "ignition":{
      "version":"2.2.0"
   },
   "passwd":{
      "users":[
         {
            "name":"giantswarm",
            "sshAuthorizedKeys":[
               "ssh-rsa AAAABEKf fake@giantswarm"
            ],
            "groups":[
               "sudo",
               "docker"
            ],
            "shell":"/bin/bash",
            "uid":1000
         }
      ]
   },
   "storage":{
      "files":[
         {
            "path":"/etc/hostname",
            "filesystem":"root",
            "mode":420,
            "contents":{
               "source":"data:,%s-bastion"
            }
         },
         {
            "path":"/etc/ssh/sshd_config",
            "filesystem":"root",
            "mode":600,
            "contents":{
               "source":"data:text/plain;charset=utf-8;base64,IyBVc2UgbW9zdCBkZWZhdWx0cyBmb3Igc3NoZCBjb25maWd1cmF0aW9uLgpTdWJzeXN0ZW0gc2Z0cCBpbnRlcm5hbC1zZnRwCkNsaWVudEFsaXZlSW50ZXJ2YWwgMTgwClVzZUROUyBubwpVc2VQQU0geWVzClByaW50TGFzdExvZyBubyAjIGhhbmRsZWQgYnkgUEFNClByaW50TW90ZCBubyAjIGhhbmRsZWQgYnkgUEFNCiMgTm9uIGRlZmF1bHRzICgjMTAwKQpDbGllbnRBbGl2ZUNvdW50TWF4IDIKUGFzc3dvcmRBdXRoZW50aWNhdGlvbiBubwpUcnVzdGVkVXNlckNBS2V5cyAvZXRjL3NzaC90cnVzdGVkLXVzZXItY2Eta2V5cy5wZW0KTWF4QXV0aFRyaWVzIDUKTG9naW5HcmFjZVRpbWUgNjAKQWxsb3dUY3BGb3J3YXJkaW5nIG5vCkFsbG93QWdlbnRGb3J3YXJkaW5nIG5vCg=="
            }
         },
         {
            "path":"/etc/ssh/trusted-user-ca-keys.pem",
            "filesystem":"root",
            "mode":400,
            "contents":{
               "source":"data:text/plain;charset=utf-8;base64,SSH_SSO_PUBLIC_KEY_PLACEHOLDER"
            }
         }
      ]
   }
}`
