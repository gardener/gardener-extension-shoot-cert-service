<p>Packages:</p>
<ul>
<li>
<a href="#shoot-cert-service.extensions.config.gardener.cloud%2fv1alpha1">shoot-cert-service.extensions.config.gardener.cloud/v1alpha1</a>
</li>
</ul>
<h2 id="shoot-cert-service.extensions.config.gardener.cloud/v1alpha1">shoot-cert-service.extensions.config.gardener.cloud/v1alpha1</h2>
<p>
<p>Package v1alpha1 contains the Certificate Shoot Service extension configuration.</p>
</p>
Resource Types:
<ul><li>
<a href="#shoot-cert-service.extensions.config.gardener.cloud/v1alpha1.Configuration">Configuration</a>
</li></ul>
<h3 id="shoot-cert-service.extensions.config.gardener.cloud/v1alpha1.Configuration">Configuration
</h3>
<p>
<p>Configuration contains information about the certificate service configuration.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
shoot-cert-service.extensions.config.gardener.cloud/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>Configuration</code></td>
</tr>
<tr>
<td>
<code>issuerName</code></br>
<em>
string
</em>
</td>
<td>
<p>IssuerName is the name of the issuer.</p>
</td>
</tr>
<tr>
<td>
<code>restrictIssuer</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>RestrictIssuer restricts the ACME issuer to shoot related domains.</p>
</td>
</tr>
<tr>
<td>
<code>defaultRequestsPerDayQuota</code></br>
<em>
int32
</em>
</td>
<td>
<em>(Optional)</em>
<p>DefaultRequestsPerDayQuota restricts the certificate requests per issuer (can be overriden in issuer spec)</p>
</td>
</tr>
<tr>
<td>
<code>shootIssuers</code></br>
<em>
<a href="#shoot-cert-service.extensions.config.gardener.cloud/v1alpha1.ShootIssuers">
ShootIssuers
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ShootIssuers contains enablement for issuers on shoot cluster</p>
</td>
</tr>
<tr>
<td>
<code>acme</code></br>
<em>
<a href="#shoot-cert-service.extensions.config.gardener.cloud/v1alpha1.ACME">
ACME
</a>
</em>
</td>
<td>
<p>ACME contains ACME related configuration.</p>
</td>
</tr>
<tr>
<td>
<code>healthCheckConfig</code></br>
<em>
github.com/gardener/gardener/extensions/pkg/apis/config/v1alpha1.HealthCheckConfig
</em>
</td>
<td>
<em>(Optional)</em>
<p>HealthCheckConfig is the config for the health check controller.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="shoot-cert-service.extensions.config.gardener.cloud/v1alpha1.ACME">ACME
</h3>
<p>
(<em>Appears on:</em>
<a href="#shoot-cert-service.extensions.config.gardener.cloud/v1alpha1.Configuration">Configuration</a>)
</p>
<p>
<p>ACME holds information about the ACME issuer used for the certificate service.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>email</code></br>
<em>
string
</em>
</td>
<td>
<p>Email is the e-mail address used for the ACME issuer.</p>
</td>
</tr>
<tr>
<td>
<code>server</code></br>
<em>
string
</em>
</td>
<td>
<p>Server is the server address used for the ACME issuer.</p>
</td>
</tr>
<tr>
<td>
<code>privateKey</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>PrivateKey is the key used for the ACME issuer.</p>
</td>
</tr>
<tr>
<td>
<code>propagationTimeout</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#duration-v1-meta">
Kubernetes meta/v1.Duration
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>PropagationTimeout is the timeout for DNS01 challenges.</p>
</td>
</tr>
<tr>
<td>
<code>precheckNameservers</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>PrecheckNameservers is used to specify a comma-separated list of DNS servers for checking availability for DNS
challenge before calling ACME CA</p>
</td>
</tr>
<tr>
<td>
<code>caCertificates</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>CACertificates are custom root certificates to be made available for the cert-controller-manager</p>
</td>
</tr>
<tr>
<td>
<code>deactivateAuthorizations</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>DeactivateAuthorizations enables deactivation of authorizations after successful certificate request</p>
</td>
</tr>
</tbody>
</table>
<h3 id="shoot-cert-service.extensions.config.gardener.cloud/v1alpha1.ShootIssuers">ShootIssuers
</h3>
<p>
(<em>Appears on:</em>
<a href="#shoot-cert-service.extensions.config.gardener.cloud/v1alpha1.Configuration">Configuration</a>)
</p>
<p>
<p>ShootIssuers holds enablement for issuers on shoot cluster</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>enabled</code></br>
<em>
bool
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<hr/>
<p><em>
Generated with <a href="https://github.com/ahmetb/gen-crd-api-reference-docs">gen-crd-api-reference-docs</a>
</em></p>
