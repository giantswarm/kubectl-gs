package key

const AppCRTemplate = `
{{- if .UserConfigConfigMap -}}
---
{{ .UserConfigConfigMap -}}
{{- end -}}
{{- if .UserConfigSecret -}}
---
{{ .UserConfigSecret -}}
{{- end -}}
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

const ClusterEKSCRsTemplate = `
{{- .ClusterCR -}}
---
{{ .AWSManagedControlPlaneCR -}}
---
{{ .AWSClusterRoleIdentityCR -}}
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

const MachinePoolEKSCRsTemplate = `
{{- .ManagedMachinePoolCR -}}
---
{{ .MachinePoolCR -}}
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
   "systemd":{
     "units":[
       {{- $units := .SystemdUnits }}
       {{- range $index,$unit := $units }}
       {{- if $index}},{{- end}}
       {
          "name":"{{ $unit.Name }}",
          "enabled":true,
          "contents": "{{ $unit.Contents }}"
       }
       {{- end }}
     ]
   },
   "storage":{
      "files":[
         {
            "path":"/etc/hostname",
            "filesystem":"root",
            "mode": 420,
            "contents":{
               "source":"data:,%s-bastion"
            }
         },
         {
            "path":"/etc/ssh/sshd_config",
            "filesystem":"root",
            "mode": 420,
            "contents":{
               "source":"data:text/plain;charset=utf-8;base64,%s"
            }
         },
         {
            "path":"/etc/ssh/trusted-user-ca-keys.pem",
            "filesystem":"root",
            "mode": 420,
            "contents":{
               "source":"data:text/plain;charset=utf-8;base64,%s"
            }
         }
      ]
   }
}`

const ubuntuSudoersConfig = "giantswarm ALL=(ALL:ALL) NOPASSWD: ALL"

const bastionSSHDConfig = `# Use most defaults for sshd configuration.
Subsystem sftp internal-sftp
ClientAliveInterval 180
UseDNS no
UsePAM yes
PrintLastLog no # handled by PAM
PrintMotd no # handled by PAM
# Non defaults (#100)
ClientAliveCountMax 2
PasswordAuthentication no
TrustedUserCAKeys /etc/ssh/trusted-user-ca-keys.pem
MaxAuthTries 5
LoginGraceTime 60
AllowTcpForwarding yes
AllowAgentForwarding yes
CASignatureAlgorithms ecdsa-sha2-nistp256,ecdsa-sha2-nistp384,ecdsa-sha2-nistp521,ssh-ed25519,rsa-sha2-512,rsa-sha2-256,ssh-rsa`

const nodeSSHDConfig = `# Use most defaults for sshd configuration.
Subsystem sftp internal-sftp
ClientAliveInterval 180
UseDNS no
UsePAM yes
PrintLastLog no # handled by PAM
PrintMotd no # handled by PAM
# Non defaults (#100)
ClientAliveCountMax 2
PasswordAuthentication no
TrustedUserCAKeys /etc/ssh/trusted-user-ca-keys.pem
MaxAuthTries 5
LoginGraceTime 60
AllowTcpForwarding no
AllowAgentForwarding no
`

const CapzSetBastionReadyTimer = `[Timer]
OnCalendar=minutely
Unit=set-bastion-ready.service

[Install]
WantedBy=timers.target
`

const CapzSetBastionReadyService = `[Unit]
Description=Set Bastion zone as ready by creating /run/cluster-api/bootstrap-success.complete

[Service]
Type=oneshot
ExecStart=/bin/sh -c 'mkdir -p /run/cluster-api/ ; echo "success" >/run/cluster-api/bootstrap-success.complete'
`

const OpenStackUserconfigTemplate = `cloudProviderOpenstack:
  configMap:
    values: |
      openstack-cinder-csi:
        clusterID: giant_swarm_cluster_{{ .ManagementCluster }}_{{ .ClusterName }}
      openstack-cloud-controller-manager:
        controllerExtraArgs: |-
          - --cluster-name=giant_swarm_cluster_{{ .ManagementCluster }}_{{ .ClusterName }}
netExporter:
  configMap:
    values: |
      dns:
        service: kube-dns
`
