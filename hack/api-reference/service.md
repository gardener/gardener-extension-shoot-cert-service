<p>Packages:</p>
<ul>
<li>
<a href="#service.cert.extensions.gardener.cloud%2fv1alpha1">service.cert.extensions.gardener.cloud/v1alpha1</a>
</li>
</ul>
<h2 id="service.cert.extensions.gardener.cloud/v1alpha1">service.cert.extensions.gardener.cloud/v1alpha1</h2>
<p>
<p>Package v1alpha1 contains the Certificate Shoot Service extension.</p>
</p>
Resource Types:
<ul><li>
<a href="#service.cert.extensions.gardener.cloud/v1alpha1.CertConfig">CertConfig</a>
</li></ul>
<h3 id="service.cert.extensions.gardener.cloud/v1alpha1.CertConfig">CertConfig
</h3>
<p>
<p>CertConfig infrastructure configuration resource</p>
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
service.cert.extensions.gardener.cloud/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>CertConfig</code></td>
</tr>
<tr>
<td>
<code>issuers</code></br>
<em>
<a href="#service.cert.extensions.gardener.cloud/v1alpha1.IssuerConfig">
[]IssuerConfig
</a>
</em>
</td>
<td>
<p>Issuers is the configuration for certificate issuers.</p>
</td>
</tr>
<tr>
<td>
<code>dnsChallengeOnShoot</code></br>
<em>
<a href="#service.cert.extensions.gardener.cloud/v1alpha1.DNSChallengeOnShoot">
DNSChallengeOnShoot
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>DNSChallengeOnShoot controls where the DNS entries for DNS01 challenges are created.
If not specified the DNS01 challenges are written to the control plane namespace on the seed.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="service.cert.extensions.gardener.cloud/v1alpha1.DNSChallengeOnShoot">DNSChallengeOnShoot
</h3>
<p>
(<em>Appears on:</em>
<a href="#service.cert.extensions.gardener.cloud/v1alpha1.CertConfig">CertConfig</a>)
</p>
<p>
<p>DNSChallengeOnShoot is used to create DNS01 challenges on shoot and not on seed.</p>
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
<tr>
<td>
<code>namespace</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>dnsClass</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="service.cert.extensions.gardener.cloud/v1alpha1.IssuerConfig">IssuerConfig
</h3>
<p>
(<em>Appears on:</em>
<a href="#service.cert.extensions.gardener.cloud/v1alpha1.CertConfig">CertConfig</a>)
</p>
<p>
<p>IssuerConfig contains information for certificate issuers.</p>
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
<code>name</code></br>
<em>
string
</em>
</td>
<td>
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
</td>
</tr>
<tr>
<td>
<code>email</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<hr/>
