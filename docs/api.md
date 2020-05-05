<div id="preamble">
<div class="sectionbody">
<div class="ulist">
<div class="title">Packages</div>
<ul>
<li>
<p><a href="#k8s-api-secrets-contentful-com-v1">secrets.contentful.com/v1</a></p>
</li>
</ul>
</div>
</div>
</div>
<div class="sect1">
<h2 id="k8s-api-secrets-contentful-com-v1">secrets.contentful.com/v1</h2>
<div class="sectionbody">
<div class="paragraph">
<p>Package v1 contains API Schema definitions for the secrets v1 API group</p>
</div>
<div class="ulist">
<div class="title">Resource Types</div>
<ul>
<li>
<p><a href="#k8s-api-github-com-contentful-labs-k8s-secret-syncer-api-v1-syncedsecret">SyncedSecret</a></p>
</li>
</ul>
</div>
<div class="sect2">
<h3 id="k8s-api-github-com-contentful-labs-k8s-secret-syncer-api-v1-datafrom">DataFrom</h3>
<div class="paragraph">
<p>DataFrom data from</p>
</div>
<div class="sidebarblock">
<div class="content">
<div class="title">Appears In:</div>
<div class="ulist">
<ul>
<li>
<p><a href="#k8s-api-github-com-contentful-labs-k8s-secret-syncer-api-v1-syncedsecretspec">SyncedSecretSpec</a></p>
</li>
</ul>
</div>
</div>
</div>
<table class="tableblock frame-all grid-all stretch">
<colgroup>
<col style="width: 25%;"/>
<col style="width: 75%;"/>
</colgroup>
<thead>
<tr>
<th class="tableblock halign-left valign-top">Field</th>
<th class="tableblock halign-left valign-top">Description</th>
</tr>
</thead>
<tbody>
<tr>
<td class="tableblock halign-left valign-top"><div class="content"><div class="paragraph">
<p><strong><code>secretRef</code></strong> <em><a href="#k8s-api-github-com-contentful-labs-k8s-secret-syncer-api-v1-secretref">SecretRef</a></em></p>
</div></div></td>
<td class="tableblock halign-left valign-top"><div class="content"><div class="paragraph">
<p>SecretRef</p>
</div></div></td>
</tr>
</tbody>
</table>
<hr/>
</div>
<div class="sect2">
<h3 id="k8s-api-github-com-contentful-labs-k8s-secret-syncer-api-v1-secretfield">SecretField</h3>
<div class="paragraph">
<p>SecretField secret field</p>
</div>
<div class="sidebarblock">
<div class="content">
<div class="title">Appears In:</div>
<div class="ulist">
<ul>
<li>
<p><a href="#k8s-api-github-com-contentful-labs-k8s-secret-syncer-api-v1-syncedsecretspec">SyncedSecretSpec</a></p>
</li>
</ul>
</div>
</div>
</div>
<table class="tableblock frame-all grid-all stretch">
<colgroup>
<col style="width: 25%;"/>
<col style="width: 75%;"/>
</colgroup>
<thead>
<tr>
<th class="tableblock halign-left valign-top">Field</th>
<th class="tableblock halign-left valign-top">Description</th>
</tr>
</thead>
<tbody>
<tr>
<td class="tableblock halign-left valign-top"><div class="content"><div class="paragraph">
<p><strong><code>name</code></strong> <em>string</em></p>
</div></div></td>
<td class="tableblock halign-left valign-top"><div class="content"><div class="paragraph">
<p>secret Name</p>
</div></div></td>
</tr>
<tr>
<td class="tableblock halign-left valign-top"><div class="content"><div class="paragraph">
<p><strong><code>value</code></strong> <em>string</em></p>
</div></div></td>
<td class="tableblock halign-left valign-top"><div class="content"><div class="paragraph">
<p>Value</p>
</div></div></td>
</tr>
<tr>
<td class="tableblock halign-left valign-top"><div class="content"><div class="paragraph">
<p><strong><code>valueFrom</code></strong> <em><a href="#k8s-api-github-com-contentful-labs-k8s-secret-syncer-api-v1-valuefrom">ValueFrom</a></em></p>
</div></div></td>
<td class="tableblock halign-left valign-top"><div class="content"><div class="paragraph">
<p>ValueFrom</p>
</div></div></td>
</tr>
</tbody>
</table>
<hr/>
</div>
<div class="sect2">
<h3 id="k8s-api-github-com-contentful-labs-k8s-secret-syncer-api-v1-secretkeyref">SecretKeyRef</h3>
<div class="paragraph">
<p>SecretKeyRef secret key ref</p>
</div>
<div class="sidebarblock">
<div class="content">
<div class="title">Appears In:</div>
<div class="ulist">
<ul>
<li>
<p><a href="#k8s-api-github-com-contentful-labs-k8s-secret-syncer-api-v1-valuefrom">ValueFrom</a></p>
</li>
</ul>
</div>
</div>
</div>
<table class="tableblock frame-all grid-all stretch">
<colgroup>
<col style="width: 25%;"/>
<col style="width: 75%;"/>
</colgroup>
<thead>
<tr>
<th class="tableblock halign-left valign-top">Field</th>
<th class="tableblock halign-left valign-top">Description</th>
</tr>
</thead>
<tbody>
<tr>
<td class="tableblock halign-left valign-top"><div class="content"><div class="paragraph">
<p><strong><code>name</code></strong> <em>string</em></p>
</div></div></td>
<td class="tableblock halign-left valign-top"><div class="content"><div class="paragraph">
<p>Secret Name</p>
</div></div></td>
</tr>
<tr>
<td class="tableblock halign-left valign-top"><div class="content"><div class="paragraph">
<p><strong><code>key</code></strong> <em>string</em></p>
</div></div></td>
<td class="tableblock halign-left valign-top"><div class="content"><div class="paragraph">
<p>Secret Key</p>
</div></div></td>
</tr>
</tbody>
</table>
<hr/>
</div>
<div class="sect2">
<h3 id="k8s-api-github-com-contentful-labs-k8s-secret-syncer-api-v1-secretref">SecretRef</h3>
<div class="paragraph">
<p>SecretRef secret ref</p>
</div>
<div class="sidebarblock">
<div class="content">
<div class="title">Appears In:</div>
<div class="ulist">
<ul>
<li>
<p><a href="#k8s-api-github-com-contentful-labs-k8s-secret-syncer-api-v1-datafrom">DataFrom</a></p>
</li>
<li>
<p><a href="#k8s-api-github-com-contentful-labs-k8s-secret-syncer-api-v1-valuefrom">ValueFrom</a></p>
</li>
</ul>
</div>
</div>
</div>
<table class="tableblock frame-all grid-all stretch">
<colgroup>
<col style="width: 25%;"/>
<col style="width: 75%;"/>
</colgroup>
<thead>
<tr>
<th class="tableblock halign-left valign-top">Field</th>
<th class="tableblock halign-left valign-top">Description</th>
</tr>
</thead>
<tbody>
<tr>
<td class="tableblock halign-left valign-top"><div class="content"><div class="paragraph">
<p><strong><code>name</code></strong> <em>string</em></p>
</div></div></td>
<td class="tableblock halign-left valign-top"><div class="content"><div class="paragraph">
<p>Secret Name</p>
</div></div></td>
</tr>
</tbody>
</table>
<hr/>
</div>
<div class="sect2">
<h3 id="k8s-api-github-com-contentful-labs-k8s-secret-syncer-api-v1-syncedsecret">SyncedSecret</h3>
<div class="paragraph">
<p>SyncedSecret is the Schema for the SyncedSecrets API</p>
</div>
<table class="tableblock frame-all grid-all stretch">
<colgroup>
<col style="width: 25%;"/>
<col style="width: 75%;"/>
</colgroup>
<thead>
<tr>
<th class="tableblock halign-left valign-top">Field</th>
<th class="tableblock halign-left valign-top">Description</th>
</tr>
</thead>
<tbody>
<tr>
<td class="tableblock halign-left valign-top"><div class="content"><div class="paragraph">
<p><strong><code>apiVersion</code></strong> <em>string</em></p>
</div></div></td>
<td class="tableblock halign-left valign-top"><div class="content"><div class="paragraph">
<p><code>secrets.contentful.com/v1</code></p>
</div></div></td>
</tr>
<tr>
<td class="tableblock halign-left valign-top"><div class="content"><div class="paragraph">
<p><strong><code>kind</code></strong> <em>string</em></p>
</div></div></td>
<td class="tableblock halign-left valign-top"><div class="content"><div class="paragraph">
<p><code>SyncedSecret</code></p>
</div></div></td>
</tr>
<tr>
<td class="tableblock halign-left valign-top"><div class="content"><div class="paragraph">
<p><strong><code>metadata</code></strong> <em><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.16/#objectmeta-v1-meta">ObjectMeta</a></em></p>
</div></div></td>
<td class="tableblock halign-left valign-top"><div class="content"><div class="paragraph">
<p>Refer to Kubernetes API documentation for fields of <code>metadata</code>.</p>
</div></div></td>
</tr>
<tr>
<td class="tableblock halign-left valign-top"><div class="content"><div class="paragraph">
<p><strong><code>spec</code></strong> <em><a href="#k8s-api-github-com-contentful-labs-k8s-secret-syncer-api-v1-syncedsecretspec">SyncedSecretSpec</a></em></p>
</div></div></td>
<td class="tableblock halign-left valign-top"><div class="content"></div></td>
</tr>
</tbody>
</table>
<hr/>
</div>
<div class="sect2">
<h3 id="k8s-api-github-com-contentful-labs-k8s-secret-syncer-api-v1-syncedsecretspec">SyncedSecretSpec</h3>
<div class="paragraph">
<p>SyncedSecretSpec defines the desired state of SyncedSecret</p>
</div>
<div class="sidebarblock">
<div class="content">
<div class="title">Appears In:</div>
<div class="ulist">
<ul>
<li>
<p><a href="#k8s-api-github-com-contentful-labs-k8s-secret-syncer-api-v1-syncedsecret">SyncedSecret</a></p>
</li>
</ul>
</div>
</div>
</div>
<table class="tableblock frame-all grid-all stretch">
<colgroup>
<col style="width: 25%;"/>
<col style="width: 75%;"/>
</colgroup>
<thead>
<tr>
<th class="tableblock halign-left valign-top">Field</th>
<th class="tableblock halign-left valign-top">Description</th>
</tr>
</thead>
<tbody>
<tr>
<td class="tableblock halign-left valign-top"><div class="content"><div class="paragraph">
<p><strong><code>secretMetadata</code></strong> <em><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.16/#objectmeta-v1-meta">ObjectMeta</a></em></p>
</div></div></td>
<td class="tableblock halign-left valign-top"><div class="content"><div class="paragraph">
<p>Secret Metadata</p>
</div></div></td>
</tr>
<tr>
<td class="tableblock halign-left valign-top"><div class="content"><div class="paragraph">
<p><strong><code>IAMRole</code></strong> <em>string</em></p>
</div></div></td>
<td class="tableblock halign-left valign-top"><div class="content"><div class="paragraph">
<p>IAMRole</p>
</div></div></td>
</tr>
<tr>
<td class="tableblock halign-left valign-top"><div class="content"><div class="paragraph">
<p><strong><code>data</code></strong> <em><a href="#k8s-api-github-com-contentful-labs-k8s-secret-syncer-api-v1-secretfield">SecretField</a></em></p>
</div></div></td>
<td class="tableblock halign-left valign-top"><div class="content"><div class="paragraph">
<p>Data</p>
</div></div></td>
</tr>
<tr>
<td class="tableblock halign-left valign-top"><div class="content"><div class="paragraph">
<p><strong><code>dataFrom</code></strong> <em><a href="#k8s-api-github-com-contentful-labs-k8s-secret-syncer-api-v1-datafrom">DataFrom</a></em></p>
</div></div></td>
<td class="tableblock halign-left valign-top"><div class="content"><div class="paragraph">
<p>DataFrom</p>
</div></div></td>
</tr>
</tbody>
</table>
<hr/>
<hr/>
</div>
<div class="sect2">
<h3 id="k8s-api-github-com-contentful-labs-k8s-secret-syncer-api-v1-valuefrom">ValueFrom</h3>
<div class="paragraph">
<p>ValueFrom value from</p>
</div>
<div class="sidebarblock">
<div class="content">
<div class="title">Appears In:</div>
<div class="ulist">
<ul>
<li>
<p><a href="#k8s-api-github-com-contentful-labs-k8s-secret-syncer-api-v1-secretfield">SecretField</a></p>
</li>
</ul>
</div>
</div>
</div>
<table class="tableblock frame-all grid-all stretch">
<colgroup>
<col style="width: 25%;"/>
<col style="width: 75%;"/>
</colgroup>
<thead>
<tr>
<th class="tableblock halign-left valign-top">Field</th>
<th class="tableblock halign-left valign-top">Description</th>
</tr>
</thead>
<tbody>
<tr>
<td class="tableblock halign-left valign-top"><div class="content"><div class="paragraph">
<p><strong><code>secretRef</code></strong> <em><a href="#k8s-api-github-com-contentful-labs-k8s-secret-syncer-api-v1-secretref">SecretRef</a></em></p>
</div></div></td>
<td class="tableblock halign-left valign-top"><div class="content"><div class="paragraph">
<p>SecretRef</p>
</div></div></td>
</tr>
<tr>
<td class="tableblock halign-left valign-top"><div class="content"><div class="paragraph">
<p><strong><code>secretKeyRef</code></strong> <em><a href="#k8s-api-github-com-contentful-labs-k8s-secret-syncer-api-v1-secretkeyref">SecretKeyRef</a></em></p>
</div></div></td>
<td class="tableblock halign-left valign-top"><div class="content"><div class="paragraph">
<p>SecretKeyRef</p>
</div></div></td>
</tr>
<tr>
<td class="tableblock halign-left valign-top"><div class="content"><div class="paragraph">
<p><strong><code>template</code></strong> <em>string</em></p>
</div></div></td>
<td class="tableblock halign-left valign-top"><div class="content"><div class="paragraph">
<p>Template</p>
</div></div></td>
</tr>
</tbody>
</table>
<hr/>
</div>
</div>
</div>