provider "pks" {
  skip_ssl_validation = true
}

provider "aws" {}

resource "pks_cluster" "example" {
  name = var.cluster_name
  external_hostname = "${var.cluster_name}.${var.k8s_api_dns_suffix}"
  plan = "small"
  num_nodes = 1
}

resource "aws_lb" "k8s_api" {
  name                             = "${pks_cluster.example.name}-api"
  load_balancer_type               = "network"
  enable_cross_zone_load_balancing = true
  internal                         = false
  subnets                          = ["${var.public_subnet_ids}"]
}

resource "aws_lb_listener" "k8s_api_8443" {
  load_balancer_arn = aws_lb.k8s_api.arn
  port              = 8443
  protocol          = "TCP"

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.k8s_api_8443.arn
  }
}

resource "aws_lb_target_group" "k8s_api_8443" {
  name     = "${pks_cluster.example.name}-k8s-tg-8443"
  port     = 8443
  protocol = "TCP"
  vpc_id   = "${var.vpc_id}"
  target_type = "ip"
}

resource "aws_lb_target_group_attachment" "k8s_api_nodes" {
  for_each = pks_cluster.example.master_ips
  target_group_arn = "${aws_lb_target_group.k8s_api_8443.arn}"
  target_id        = each.value
  port             = 80
}

resource "aws_route53_record" "api" {
  zone_id = "${var.zone_id}"
  name = "${pks_cluster.example.external_hostname}"
  value = "${aws_lb.k8s_api.dns_name}"
  type = "CNAME"
}

variable "cluster_name" {
  type = "string"
  default = "example-tf"
}


variable "k8s_api_dns_suffix" {
  type = "string"
}


variable "public_subnet_ids" {
  type = "list"
}

variable "vpc_id" {
  type = "string"
}

variable "zone_id" {
  type = "string"
}
