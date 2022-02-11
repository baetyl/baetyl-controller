bash $GOPATH/src/k8s.io/code-generator/generate-groups.sh \
  "deepcopy,client,informer,lister" \
  github.com/baetyl/baetyl-controller/kube/client \
  github.com/baetyl/baetyl-controller/kube/apis \
  "baetyl:v1alpha1"

# 注意 code-generator 库的版本要和 go.mod 中引用的 k8s 库保持一致
# 期望生成的函数列表 deepcopy,defaulter,client,lister,informer
# 生成代码的目标目录
# CRD 所在目录
# CRD的group name和version