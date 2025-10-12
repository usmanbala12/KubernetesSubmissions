# KubernetesSubmissions

## Exercises
### chapter 2
- [1.1](https://github.com/usmanbala12/KubernetesSubmissions/tree/1.1/log_output)
- [1.2](https://github.com/usmanbala12/KubernetesSubmissions/tree/1.2/todoapp)
- [1.3](https://github.com/usmanbala12/KubernetesSubmissions/tree/1.3/log_output)
- [1.4](https://github.com/usmanbala12/KubernetesSubmissions/tree/1.4/todoapp)
- [1.5](https://github.com/usmanbala12/KubernetesSubmissions/tree/1.5/todoapp)
- [1.6](https://github.com/usmanbala12/KubernetesSubmissions/tree/1.6/todoapp)
- [1.7](https://github.com/usmanbala12/KubernetesSubmissions/tree/1.7/log_output)
- [1.8](https://github.com/usmanbala12/KubernetesSubmissions/tree/1.8/todoapp)
- [1.9](https://github.com/usmanbala12/KubernetesSubmissions/tree/1.9/ping-pong)
- [1.10](https://github.com/usmanbala12/KubernetesSubmissions/tree/1.10/log_output)
- [1.11](https://github.com/usmanbala12/KubernetesSubmissions/tree/1.11)
- [1.12](https://github.com/usmanbala12/KubernetesSubmissions/tree/1.12/todoapp)
- [1.13](https://github.com/usmanbala12/KubernetesSubmissions/tree/1.13/todoapp)

### chapter 3
- [2.1](https://github.com/usmanbala12/KubernetesSubmissions/tree/2.1/ping-pong)
- [2.2](https://github.com/usmanbala12/KubernetesSubmissions/tree/2.2/todoapp)
- [2.3](https://github.com/usmanbala12/KubernetesSubmissions/tree/2.3)
- [2.4](https://github.com/usmanbala12/KubernetesSubmissions/tree/2.4/todoapp)
- [2.5](https://github.com/usmanbala12/KubernetesSubmissions/tree/2.5/log_output)
- [2.6](https://github.com/usmanbala12/KubernetesSubmissions/tree/2.6/log_output)
- [2.7](https://github.com/usmanbala12/KubernetesSubmissions/tree/2.7/ping-pong)
- [2.8](https://github.com/usmanbala12/KubernetesSubmissions/tree/2.8/todoapp)
- [2.9](https://github.com/usmanbala12/KubernetesSubmissions/tree/2.9/todoapp/manifests)
- [2.10](https://github.com/usmanbala12/KubernetesSubmissions/tree/2.10/todoapp/todo-backend)

### chapter 4
- [3.1](https://github.com/usmanbala12/KubernetesSubmissions/tree/3.1/ping-pong)
- [3.2](https://github.com/usmanbala12/KubernetesSubmissions/tree/3.2)
- [3.3](https://github.com/usmanbala12/KubernetesSubmissions/tree/3.3/log_output/manifests)
- [3.4](https://github.com/usmanbala12/KubernetesSubmissions/tree/3.4/ping-pong/manifests)
- [3.5](https://github.com/usmanbala12/KubernetesSubmissions/tree/3.5/todoapp)
- [3.6](https://github.com/usmanbala12/KubernetesSubmissions/tree/3.6/.github/workflows)
- [3.7](https://github.com/usmanbala12/KubernetesSubmissions/tree/3.7/.github/workflows)
- [3.8](https://github.com/usmanbala12/KubernetesSubmissions/tree/3.8)

### 3.9. DBaaS vs DIY
Database Deployment: DBaaS vs. Kubernetes
1. Required Work and Cost to Initialize

DBaass **Pros**

* **Easy Setup:** Provision with a few clicks or an API call.
* **Low Initial Cost:** Pay-as-you-go; no hardware or licensing fees.
* **No Expertise Needed:** No deep DBA or infra knowledge required.

**Cons**

* **Higher Long-Term Cost:** More expensive at scale than self-managed.
* **Limited Configuration:** Versions, extensions, and hardware are provider-defined.

Kubernetes **Pros**

* **Lower Long-Term Cost:** Can be cheaper at scale without managed-service premiums.
* **Full Control:** Customize versions, configs, and infrastructure.

**Cons**

* **High Setup Effort:** Production-ready clusters with storage are complex.
* **Expertise Required:** Requires Kubernetes, storage, and DB admin skills.
* **Higher Upfront Cost:** Infra setup (on-prem or cloud) can be costly.

2. Ongoing Maintenance

DBaaS **Pros**

* **Automated Maintenance:** Provider manages patches, updates, scaling.
* **Lower Ops Load:** Team focuses on application development.
* **Built-in Availability:** High availability and SLOs included.

**Cons**

* **No Control Over Timing:** Maintenance schedules may not align with your needs.
* **Limited Troubleshooting:** Reliant on provider support with less infra visibility.

Kubernetes **Pros**

* **Full Maintenance Control:** You choose timing and methods.
* **Deep Visibility:** Access all stack layers for tuning and troubleshooting.

**Cons**

* **Heavy Ops Burden:** Team handles updates, scaling, security, and recovery.
* **Complex Stateful Management:** Databases are harder to run in Kubernetes.
* **No SLA:** You must ensure uptime and disaster recovery.

3. Backup Methods and Ease of Use

DBaaS **Pros**

* **Automated Backups:** Enabled by default on a schedule.
* **Point-in-Time Recovery:** Restore to any second.
* **Simple Restores:** One-click via console or API.
* **Managed Offsite Storage:** Safe from regional outages.

**Cons**

* **Limited Customization:** Frequency, retention, and location may be fixed.
* **Extra Costs:** Storage and retrieval may incur fees.

Kubernetes **Pros**

* **Full Flexibility:** Choose tools and strategies
* **Cost Optimization:** Tailor strategy to minimize storage costs.

**Cons**

* **Complex Setup:** Automating reliable backups requires expertise.
* **Manual Restores:** Often multi-step and error-prone.
* **Misconfiguration Risk:** Higher chance of human error causing data loss.

-[3.9](https://github.com/usmanbala12/KubernetesSubmissions/tree/3.19)
-[3.10](https://github.com/usmanbala12/KubernetesSubmissions/tree/3.10/todoapp/pg-backup)
-[3.11](https://github.com/usmanbala12/KubernetesSubmissions/tree/3.11/todoapp/manifests)
-[3.12](https://github.com/usmanbala12/KubernetesSubmissions/tree/3.12/todoapp)

-[4.1](https://github.com/usmanbala12/KubernetesSubmissions/tree/4.1)
-[4.2](https://github.com/usmanbala12/KubernetesSubmissions/tree/4.2/todoapp)
-[4.3](https://github.com/usmanbala12/KubernetesSubmissions/tree/4.3)
-[4.4](https://github.com/usmanbala12/KubernetesSubmissions/tree/4.4)
-[4.5](https://github.com/usmanbala12/KubernetesSubmissions/tree/4.5/todoapp)
-[4.6](https://github.com/usmanbala12/KubernetesSubmissions/tree/4.6/todoapp/broadcaster)
-[4.7](https://github.com/usmanbala12/KubernetesSubmissions/tree/4.7/log_output)
-[4.8](https://github.com/usmanbala12/KubernetesSubmissions/tree/4.8/.github/workflows)
-[4.9](https://github.com/usmanbala12/KubernetesSubmissions/tree/4.9/todoapp)

## The Project, the grande finale
-[4.10](https://github.com/usmanbala12/dwk-project-config) - config repo
-[4.10](https://github.com/usmanbala12/KubernetesSubmissions/tree/4.10)
-[5.1](https://github.com/usmanbala12/KubernetesSubmissions/tree/5.1/dummysite)
-[5.2](https://github.com/usmanbala12/KubernetesSubmissions/tree/5.2/istio-getting-started)
-[5.3](https://github.com/usmanbala12/KubernetesSubmissions/tree/5.3/log_output)
-[5.4](https://github.com/usmanbala12/KubernetesSubmissions/tree/5.4/wikipedia_init_and_sidercar)
-[5.6](https://github.com/usmanbala12/KubernetesSubmissions/tree/5.6/serverless)
-[5.7](https://github.com/usmanbala12/KubernetesSubmissions/tree/5.7/ping-pong)

