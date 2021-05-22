需要修改的是api/v1下的elasticweb_types.go文件和controller文件夹下的elasticweb_controller.go文件中的内容。 CR示例在config/samples文件夹下



webhook-operator:v3是加入了mutatingwebhook资源，并且解决了之前auditwebhook pod创建失败的问题，原因是证书并没有进行decoding



webhook-operator:v4是

