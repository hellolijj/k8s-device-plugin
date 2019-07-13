# gpu topology device plugin

## gpu topology device plugin

考虑 gpu 拓扑性的 gpu 调度

## 的安装部署

<a name="93ALF"></a>
###  配置 node 节点上容器
给每个工作节点上的容器配置支持 gpu 和 阿里云镜像下载加速器。

修改文件/etc/docker/daemon.json 如下：
```json
{
   "default-runtime": "nvidia",
   "runtimes": {
        "nvidia": {
            "path": "/usr/bin/nvidia-container-runtime",
            "runtimeArgs": []
        }
    },
   "registry-mirrors": ["https://cagz8nbe.mirror.aliyuncs.com"]
}
```

重新加载配置文件
```bash
$ systemctl daemon-reload
$ systemctl restart docker
```

检查 docker runtime<br />![image.png](https://cdn.nlark.com/yuque/0/2019/png/394957/1562773092661-01701200-756b-425a-8940-8b26fe72db40.png#align=left&display=inline&height=93&name=image.png&originHeight=186&originWidth=954&size=59640&status=done&width=477)

<a name="MKZ6D"></a>
### 普通节点上部署 device-plugin 

部署 device-plugin （ds rbac）

```bash
$ kubectl apply -f https://raw.githubusercontent.com/hellolijj/k8s-device-plugin/gsoc/deploy/gsoc-device-plugin-demo2.yaml
```

> ⚠️如果节点上已经安装了 nvidia-plugin 需要先将其删掉。如果是 static pod 需要将其一移开 /etc/kubernetes/manifest 目录。

<a name="vQNPY"></a>
### 给节点打标签使支持gpu topologyl

```yaml
$ kubectl label node <target_node> gputopology=true
```

出现如下情况，则部署成功。<br />![image.png](https://cdn.nlark.com/yuque/0/2019/png/394957/1562761440914-7b362d10-b3af-46cb-8dde-a63c2aa192d6.png#align=left&display=inline&height=75&name=image.png&originHeight=150&originWidth=2300&size=88389&status=done&width=1150)
