// Kubernetes Master Dashboard Application Logic
let currentTab = 'overview';
let currentWorkloadSubTab = 'pods';
let currentNetSubTab = 'services';
let currentStorageSubTab = 'configmaps';
let currentNamespace = '';
let autoRefreshTimer = null;
let currentLogInterval = null;
let activeLogPod = null;
let activeLogNamespace = null;
let activeLogContainer = null;

// Initialize on page load
document.addEventListener('DOMContentLoaded', () => {
    checkClusterStatus();
    loadNamespaces();
    switchTab('overview');
    setupEventListeners();
    setupAutoRefresh();
});

function setupEventListeners() {
    // Namespace change
    document.getElementById('namespaceSelect').addEventListener('change', (e) => {
        currentNamespace = e.target.value;
        refreshActiveTab();
    });

    // Auto-refresh interval change
    document.getElementById('refreshInterval').addEventListener('change', (e) => {
        setupAutoRefresh(parseInt(e.target.value, 10));
    });

    // Search filter
    document.getElementById('searchInput').addEventListener('input', (e) => {
        filterTable(e.target.value.toLowerCase());
    });

    // AI prompt enter key
    const aiInput = document.getElementById('aiPromptInput');
    if (aiInput) {
        aiInput.addEventListener('keydown', (e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                sendAIMessage();
            }
        });
    }
}

function setupAutoRefresh(seconds = 10) {
    if (autoRefreshTimer) {
        clearInterval(autoRefreshTimer);
        autoRefreshTimer = null;
    }
    if (seconds > 0) {
        autoRefreshTimer = setInterval(() => {
            refreshActiveTab(true);
        }, seconds * 1000);
    }
}

// Check cluster connection status
async function checkClusterStatus() {
    try {
        const res = await fetch('/api/status');
        const data = await res.json();
        const badge = document.getElementById('connectionBadge');
        const text = document.getElementById('connectionText');
        const dot = document.getElementById('connectionDot');

        if (data.connected) {
            badge.className = 'flex items-center gap-2 px-3 py-1.5 rounded-full text-xs font-semibold bg-emerald-500/10 text-emerald-400 border border-emerald-500/20';
            text.textContent = `Connected (${data.version || 'v1.x'})`;
            dot.className = 'w-2 h-2 rounded-full bg-emerald-400 pulse-green';
            document.getElementById('disconnectAlert')?.classList.add('hidden');
        } else {
            badge.className = 'flex items-center gap-2 px-3 py-1.5 rounded-full text-xs font-semibold bg-rose-500/10 text-rose-400 border border-rose-500/20 cursor-pointer';
            text.textContent = 'Disconnected (Click to Connect)';
            dot.className = 'w-2 h-2 rounded-full bg-rose-400';
            showDisconnectAlert(data.error || 'Cluster tidak terhubung');
        }
    } catch (err) {
        console.error('Failed to get status:', err);
    }
}

function showDisconnectAlert(msg) {
    const alertBox = document.getElementById('disconnectAlert');
    if (alertBox) {
        alertBox.classList.remove('hidden');
        document.getElementById('disconnectErrorMsg').textContent = msg;
    }
}

// Load namespaces for dropdown
async function loadNamespaces() {
    try {
        const res = await fetch('/api/namespaces');
        if (!res.ok) return;
        const namespaces = await res.json();
        const select = document.getElementById('namespaceSelect');
        const current = select.value;
        select.innerHTML = '<option value="">All Namespaces</option>';
        namespaces.forEach(ns => {
            const opt = document.createElement('option');
            opt.value = ns.name;
            opt.textContent = ns.name;
            if (ns.name === current) opt.selected = true;
            select.appendChild(opt);
        });
    } catch (err) {
        console.error('Failed to load namespaces', err);
    }
}

// Navigation Tab Switcher
function switchTab(tabId) {
    currentTab = tabId;
    document.querySelectorAll('.tab-btn').forEach(btn => {
        btn.classList.remove('bg-indigo-600/20', 'text-indigo-400', 'border-indigo-500/40');
        btn.classList.add('text-slate-400', 'hover:text-slate-200');
    });

    const activeBtn = document.getElementById(`tabBtn-${tabId}`);
    if (activeBtn) {
        activeBtn.classList.add('bg-indigo-600/20', 'text-indigo-400', 'border-indigo-500/40');
        activeBtn.classList.remove('text-slate-400');
    }

    document.querySelectorAll('.tab-content').forEach(section => {
        section.classList.add('hidden');
    });

    const activeSection = document.getElementById(`section-${tabId}`);
    if (activeSection) {
        activeSection.classList.remove('hidden');
    }

    refreshActiveTab();
}

function refreshActiveTab(isBackground = false) {
    switch (currentTab) {
        case 'overview':
            loadOverview(isBackground);
            break;
        case 'nodes':
            loadNodes(isBackground);
            break;
        case 'workloads':
            loadWorkloads(isBackground);
            break;
        case 'networking':
            loadNetworking(isBackground);
            break;
        case 'storage':
            loadStorage(isBackground);
            break;
        case 'events':
            loadEvents(isBackground);
            break;
        case 'ai':
            loadAIConfigInfo();
            break;
        case 'settings':
            loadSettingsInfo();
            break;
    }
}

// ----------------- OVERVIEW -----------------
async function loadOverview(isBackground = false) {
    const loader = document.getElementById('overviewLoader');
    if (!isBackground && loader) loader.classList.remove('hidden');

    try {
        const res = await fetch('/api/overview');
        if (!res.ok) throw new Error('Cluster not connected');
        const data = await res.json();

        document.getElementById('statNodes').textContent = `${data.readyNodes} / ${data.totalNodes}`;
        document.getElementById('statPods').textContent = data.totalPods;
        document.getElementById('statPodsRunning').textContent = `${data.runningPods} Running`;
        document.getElementById('statPodsPending').textContent = `${data.pendingPods} Pending`;
        document.getElementById('statDeployments').textContent = data.totalDeployments;
        document.getElementById('statServices').textContent = data.totalServices;
        document.getElementById('statNamespaces').textContent = data.totalNamespaces;
        document.getElementById('statEvents').textContent = data.warningEventsCount;

        document.getElementById('clusterVersion').textContent = data.kubernetesVersion || '-';
        document.getElementById('clusterAPIServer').textContent = data.apiServerUrl || '-';
        document.getElementById('clusterConnectedVia').textContent = data.connectedVia || '-';
        document.getElementById('clusterCPUCapacity').textContent = data.cpuCapacity || '-';
        document.getElementById('clusterMemCapacity').textContent = data.memoryCapacity || '-';
        document.getElementById('clusterCPUAlloc').textContent = data.cpuAllocatable || '-';
        document.getElementById('clusterMemAlloc').textContent = data.memoryAllocatable || '-';

        document.getElementById('disconnectAlert')?.classList.add('hidden');
    } catch (err) {
        checkClusterStatus();
    } finally {
        if (loader) loader.classList.add('hidden');
    }
}

// ----------------- NODES -----------------
async function loadNodes(isBackground = false) {
    const tbody = document.getElementById('nodesTableBody');
    if (!isBackground) tbody.innerHTML = `<tr><td colspan="7" class="text-center py-8 text-slate-400">Loading nodes...</td></tr>`;

    try {
        const res = await fetch('/api/nodes');
        if (!res.ok) throw new Error('Failed to fetch nodes');
        const nodes = await res.json();

        if (nodes.length === 0) {
            tbody.innerHTML = `<tr><td colspan="7" class="text-center py-8 text-slate-400">No nodes found</td></tr>`;
            return;
        }

        tbody.innerHTML = nodes.map(node => {
            const isReady = node.status.includes('Ready') && !node.status.includes('NotReady');
            const badgeClass = isReady ? 'badge-ready' : 'badge-notready';
            const cordonAction = node.unschedulable ? 'Uncordon' : 'Cordon';
            const cordonBtnColor = node.unschedulable ? 'hover:bg-emerald-600/30 text-emerald-400' : 'hover:bg-amber-600/30 text-amber-400';

            return `
                <tr class="border-b border-slate-800/60 hover:bg-slate-800/30 transition search-row">
                    <td class="py-3 px-4 font-semibold text-slate-200">
                        <div class="flex items-center gap-2">
                            <span class="w-2 h-2 rounded-full ${isReady ? 'bg-emerald-400' : 'bg-rose-400'}"></span>
                            <span>${escapeHtml(node.name)}</span>
                        </div>
                    </td>
                    <td class="py-3 px-4">
                        <span class="px-2.5 py-1 rounded text-xs font-medium ${badgeClass}">
                            ${escapeHtml(node.status)}
                        </span>
                    </td>
                    <td class="py-3 px-4 text-slate-300">
                        ${node.roles.map(r => `<span class="inline-block bg-slate-800 text-slate-300 border border-slate-700 px-2 py-0.5 rounded text-xs mr-1">${escapeHtml(r)}</span>`).join('')}
                    </td>
                    <td class="py-3 px-4 font-mono text-xs text-slate-300">
                        <div>Internal: ${escapeHtml(node.internalIP || '-')}</div>
                        ${node.externalIP ? `<div class="text-slate-400">External: ${escapeHtml(node.externalIP)}</div>` : ''}
                    </td>
                    <td class="py-3 px-4 text-xs text-slate-300">
                        <div>CPU: ${escapeHtml(node.cpuCapacity)}</div>
                        <div>Mem: ${escapeHtml(node.memoryCapacity)}</div>
                        <div class="text-slate-400">Pods: ${node.podsCount}</div>
                    </td>
                    <td class="py-3 px-4 text-xs text-slate-400">${escapeHtml(node.kubeletVersion)}</td>
                    <td class="py-3 px-4 text-xs text-slate-400">${escapeHtml(node.age)}</td>
                    <td class="py-3 px-4 text-right">
                        <button onclick="toggleCordon('${escapeHtml(node.name)}', ${!node.unschedulable})"
                            class="px-2.5 py-1 text-xs border border-slate-700 rounded transition ${cordonBtnColor}">
                            ${cordonAction}
                        </button>
                    </td>
                </tr>
            `;
        }).join('');
    } catch (err) {
        tbody.innerHTML = `<tr><td colspan="7" class="text-center py-8 text-rose-400">Error loading nodes: ${err.message}</td></tr>`;
    }
}

async function toggleCordon(nodeName, unschedulable) {
    const actionText = unschedulable ? 'Cordon' : 'Uncordon';
    if (!confirm(`Apakah Anda yakin ingin melakukan ${actionText} pada node ${nodeName}?`)) return;

    try {
        const res = await fetch('/api/nodes/cordon', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ nodeName, unschedulable })
        });
        const data = await res.json();
        if (!res.ok) throw new Error(data.error || 'Failed');
        showToast(data.message || `Node ${actionText} berhasil`, 'success');
        loadNodes();
    } catch (err) {
        showToast(err.message, 'error');
    }
}

// ----------------- WORKLOADS -----------------
function switchWorkloadSubTab(subTab) {
    currentWorkloadSubTab = subTab;
    document.querySelectorAll('.workload-subtab-btn').forEach(btn => {
        btn.classList.remove('border-indigo-500', 'text-indigo-400');
        btn.classList.add('border-transparent', 'text-slate-400');
    });
    document.getElementById(`subTabBtn-${subTab}`)?.classList.add('border-indigo-500', 'text-indigo-400');

    document.querySelectorAll('.workload-subtab-content').forEach(el => el.classList.add('hidden'));
    document.getElementById(`workloadContent-${subTab}`)?.classList.remove('hidden');

    loadWorkloads();
}

function loadWorkloads(isBackground = false) {
    switch (currentWorkloadSubTab) {
        case 'pods':
            loadPods(isBackground);
            break;
        case 'deployments':
            loadDeployments(isBackground);
            break;
        case 'daemonsets':
            loadDaemonSets(isBackground);
            break;
        case 'statefulsets':
            loadStatefulSets(isBackground);
            break;
    }
}

// Pods List
async function loadPods(isBackground = false) {
    const tbody = document.getElementById('podsTableBody');
    if (!isBackground) tbody.innerHTML = `<tr><td colspan="7" class="text-center py-8 text-slate-400">Loading pods...</td></tr>`;

    try {
        const url = `/api/pods?namespace=${encodeURIComponent(currentNamespace)}`;
        const res = await fetch(url);
        if (!res.ok) throw new Error('Failed to fetch pods');
        const pods = await res.json();

        if (pods.length === 0) {
            tbody.innerHTML = `<tr><td colspan="7" class="text-center py-8 text-slate-400">No pods found</td></tr>`;
            return;
        }

        tbody.innerHTML = pods.map(pod => {
            let badgeClass = 'badge-neutral';
            if (pod.status === 'Running') badgeClass = 'badge-running';
            else if (pod.status === 'Pending') badgeClass = 'badge-pending';
            else badgeClass = 'badge-failed';

            const containersJson = encodeURIComponent(JSON.stringify(pod.containers || []));

            return `
                <tr class="border-b border-slate-800/60 hover:bg-slate-800/30 transition search-row">
                    <td class="py-3 px-4 font-semibold text-slate-200">
                        <div class="text-sm">${escapeHtml(pod.name)}</div>
                        <div class="text-xs text-slate-500 font-mono">${escapeHtml(pod.ip || 'No IP')} • ${escapeHtml(pod.node || '-')}</div>
                    </td>
                    <td class="py-3 px-4 text-xs text-slate-400">${escapeHtml(pod.namespace)}</td>
                    <td class="py-3 px-4">
                        <span class="px-2.5 py-1 rounded text-xs font-medium ${badgeClass}">
                            ${escapeHtml(pod.status)}
                        </span>
                    </td>
                    <td class="py-3 px-4 text-xs text-slate-300 font-mono">${escapeHtml(pod.ready)}</td>
                    <td class="py-3 px-4 text-xs ${pod.restarts > 0 ? 'text-amber-400 font-bold' : 'text-slate-400'}">${pod.restarts}</td>
                    <td class="py-3 px-4 text-xs text-slate-400">${escapeHtml(pod.age)}</td>
                    <td class="py-3 px-4 text-right">
                        <div class="flex items-center justify-end gap-2">
                            <button onclick="openLogsModal('${escapeHtml(pod.namespace)}', '${escapeHtml(pod.name)}', '${containersJson}')"
                                class="px-2.5 py-1 text-xs bg-slate-800 hover:bg-slate-700 text-indigo-300 rounded border border-slate-700 transition">
                                Logs
                            </button>
                            <button onclick="deletePod('${escapeHtml(pod.namespace)}', '${escapeHtml(pod.name)}')"
                                class="px-2.5 py-1 text-xs bg-rose-500/10 hover:bg-rose-500/20 text-rose-400 rounded border border-rose-500/20 transition">
                                Delete
                            </button>
                        </div>
                    </td>
                </tr>
            `;
        }).join('');
    } catch (err) {
        tbody.innerHTML = `<tr><td colspan="7" class="text-center py-8 text-rose-400">Error loading pods: ${err.message}</td></tr>`;
    }
}

// Deployments List
async function loadDeployments(isBackground = false) {
    const tbody = document.getElementById('deploymentsTableBody');
    if (!isBackground) tbody.innerHTML = `<tr><td colspan="6" class="text-center py-8 text-slate-400">Loading deployments...</td></tr>`;

    try {
        const url = `/api/deployments?namespace=${encodeURIComponent(currentNamespace)}`;
        const res = await fetch(url);
        if (!res.ok) throw new Error('Failed to fetch deployments');
        const deps = await res.json();

        if (deps.length === 0) {
            tbody.innerHTML = `<tr><td colspan="6" class="text-center py-8 text-slate-400">No deployments found</td></tr>`;
            return;
        }

        tbody.innerHTML = deps.map(dep => {
            const isFullyReady = dep.readyReplicas === dep.replicas && dep.replicas > 0;
            const badgeClass = isFullyReady ? 'badge-ready' : (dep.readyReplicas > 0 ? 'badge-warning' : 'badge-neutral');

            return `
                <tr class="border-b border-slate-800/60 hover:bg-slate-800/30 transition search-row">
                    <td class="py-3 px-4 font-semibold text-slate-200">${escapeHtml(dep.name)}</td>
                    <td class="py-3 px-4 text-xs text-slate-400">${escapeHtml(dep.namespace)}</td>
                    <td class="py-3 px-4">
                        <span class="px-2.5 py-1 rounded text-xs font-mono font-medium ${badgeClass}">
                            ${dep.readyReplicas} / ${dep.replicas} Ready
                        </span>
                    </td>
                    <td class="py-3 px-4 text-xs text-slate-400 max-w-xs truncate" title="${dep.images.join(', ')}">
                        ${dep.images.map(img => `<span class="bg-slate-800 px-2 py-0.5 rounded border border-slate-700 font-mono">${escapeHtml(img)}</span>`).join(' ')}
                    </td>
                    <td class="py-3 px-4 text-xs text-slate-400">${escapeHtml(dep.age)}</td>
                    <td class="py-3 px-4 text-right">
                        <div class="flex items-center justify-end gap-2">
                            <button onclick="openScaleModal('${escapeHtml(dep.namespace)}', '${escapeHtml(dep.name)}', ${dep.replicas})"
                                class="px-2.5 py-1 text-xs bg-indigo-600/20 hover:bg-indigo-600/30 text-indigo-400 rounded border border-indigo-500/30 transition">
                                Scale
                            </button>
                            <button onclick="restartDeployment('${escapeHtml(dep.namespace)}', '${escapeHtml(dep.name)}')"
                                class="px-2.5 py-1 text-xs bg-slate-800 hover:bg-slate-700 text-slate-300 rounded border border-slate-700 transition">
                                Restart
                            </button>
                        </div>
                    </td>
                </tr>
            `;
        }).join('');
    } catch (err) {
        tbody.innerHTML = `<tr><td colspan="6" class="text-center py-8 text-rose-400">Error: ${err.message}</td></tr>`;
    }
}

// DaemonSets List
async function loadDaemonSets(isBackground = false) {
    const tbody = document.getElementById('daemonsetsTableBody');
    if (!isBackground) tbody.innerHTML = `<tr><td colspan="5" class="text-center py-8 text-slate-400">Loading daemonsets...</td></tr>`;

    try {
        const url = `/api/daemonsets?namespace=${encodeURIComponent(currentNamespace)}`;
        const res = await fetch(url);
        const list = await res.json();

        if (list.length === 0) {
            tbody.innerHTML = `<tr><td colspan="5" class="text-center py-8 text-slate-400">No daemonsets found</td></tr>`;
            return;
        }

        tbody.innerHTML = list.map(ds => `
            <tr class="border-b border-slate-800/60 hover:bg-slate-800/30 transition search-row">
                <td class="py-3 px-4 font-semibold text-slate-200">${escapeHtml(ds.name)}</td>
                <td class="py-3 px-4 text-xs text-slate-400">${escapeHtml(ds.namespace)}</td>
                <td class="py-3 px-4 text-xs font-mono text-slate-300">${ds.readyNumber} / ${ds.desiredNumber} Ready</td>
                <td class="py-3 px-4 text-xs text-slate-400 max-w-xs truncate font-mono">${escapeHtml(ds.images.join(', '))}</td>
                <td class="py-3 px-4 text-xs text-slate-400">${escapeHtml(ds.age)}</td>
            </tr>
        `).join('');
    } catch (err) {
        tbody.innerHTML = `<tr><td colspan="5" class="text-center py-8 text-rose-400">Error: ${err.message}</td></tr>`;
    }
}

// StatefulSets List
async function loadStatefulSets(isBackground = false) {
    const tbody = document.getElementById('statefulsetsTableBody');
    if (!isBackground) tbody.innerHTML = `<tr><td colspan="5" class="text-center py-8 text-slate-400">Loading statefulsets...</td></tr>`;

    try {
        const url = `/api/statefulsets?namespace=${encodeURIComponent(currentNamespace)}`;
        const res = await fetch(url);
        const list = await res.json();

        if (list.length === 0) {
            tbody.innerHTML = `<tr><td colspan="5" class="text-center py-8 text-slate-400">No statefulsets found</td></tr>`;
            return;
        }

        tbody.innerHTML = list.map(sts => `
            <tr class="border-b border-slate-800/60 hover:bg-slate-800/30 transition search-row">
                <td class="py-3 px-4 font-semibold text-slate-200">${escapeHtml(sts.name)}</td>
                <td class="py-3 px-4 text-xs text-slate-400">${escapeHtml(sts.namespace)}</td>
                <td class="py-3 px-4 text-xs font-mono text-slate-300">${sts.readyReplicas} / ${sts.replicas} Ready</td>
                <td class="py-3 px-4 text-xs text-slate-400 max-w-xs truncate font-mono">${escapeHtml(sts.images.join(', '))}</td>
                <td class="py-3 px-4 text-xs text-slate-400">${escapeHtml(sts.age)}</td>
            </tr>
        `).join('');
    } catch (err) {
        tbody.innerHTML = `<tr><td colspan="5" class="text-center py-8 text-rose-400">Error: ${err.message}</td></tr>`;
    }
}

// ----------------- NETWORKING -----------------
function switchNetSubTab(subTab) {
    currentNetSubTab = subTab;
    document.querySelectorAll('.net-subtab-btn').forEach(btn => {
        btn.classList.remove('border-indigo-500', 'text-indigo-400');
        btn.classList.add('border-transparent', 'text-slate-400');
    });
    document.getElementById(`netSubTabBtn-${subTab}`)?.classList.add('border-indigo-500', 'text-indigo-400');

    document.querySelectorAll('.net-subtab-content').forEach(el => el.classList.add('hidden'));
    document.getElementById(`netContent-${subTab}`)?.classList.remove('hidden');

    loadNetworking();
}

function loadNetworking(isBackground = false) {
    if (currentNetSubTab === 'services') loadServices(isBackground);
    else loadIngresses(isBackground);
}

async function loadServices(isBackground = false) {
    const tbody = document.getElementById('servicesTableBody');
    if (!isBackground) tbody.innerHTML = `<tr><td colspan="6" class="text-center py-8 text-slate-400">Loading services...</td></tr>`;

    try {
        const url = `/api/services?namespace=${encodeURIComponent(currentNamespace)}`;
        const res = await fetch(url);
        const services = await res.json();

        if (services.length === 0) {
            tbody.innerHTML = `<tr><td colspan="6" class="text-center py-8 text-slate-400">No services found</td></tr>`;
            return;
        }

        tbody.innerHTML = services.map(s => `
            <tr class="border-b border-slate-800/60 hover:bg-slate-800/30 transition search-row">
                <td class="py-3 px-4 font-semibold text-slate-200">${escapeHtml(s.name)}</td>
                <td class="py-3 px-4 text-xs text-slate-400">${escapeHtml(s.namespace)}</td>
                <td class="py-3 px-4 text-xs text-indigo-400 font-medium">${escapeHtml(s.type)}</td>
                <td class="py-3 px-4 text-xs font-mono text-slate-300">${escapeHtml(s.clusterIP)}</td>
                <td class="py-3 px-4 text-xs font-mono text-slate-400">${escapeHtml(s.externalIP)}</td>
                <td class="py-3 px-4 text-xs font-mono text-emerald-400">${escapeHtml(s.ports.join(', '))}</td>
                <td class="py-3 px-4 text-xs text-slate-400">${escapeHtml(s.age)}</td>
            </tr>
        `).join('');
    } catch (err) {
        tbody.innerHTML = `<tr><td colspan="6" class="text-center py-8 text-rose-400">Error: ${err.message}</td></tr>`;
    }
}

async function loadIngresses(isBackground = false) {
    const tbody = document.getElementById('ingressesTableBody');
    if (!isBackground) tbody.innerHTML = `<tr><td colspan="5" class="text-center py-8 text-slate-400">Loading ingresses...</td></tr>`;

    try {
        const url = `/api/ingresses?namespace=${encodeURIComponent(currentNamespace)}`;
        const res = await fetch(url);
        const ingresses = await res.json();

        if (ingresses.length === 0) {
            tbody.innerHTML = `<tr><td colspan="5" class="text-center py-8 text-slate-400">No ingresses found</td></tr>`;
            return;
        }

        tbody.innerHTML = ingresses.map(ing => `
            <tr class="border-b border-slate-800/60 hover:bg-slate-800/30 transition search-row">
                <td class="py-3 px-4 font-semibold text-slate-200">${escapeHtml(ing.name)}</td>
                <td class="py-3 px-4 text-xs text-slate-400">${escapeHtml(ing.namespace)}</td>
                <td class="py-3 px-4 text-xs font-mono text-indigo-400">${escapeHtml(ing.class)}</td>
                <td class="py-3 px-4 text-xs font-mono text-slate-300">${escapeHtml(ing.hosts.join(', ') || '*')}</td>
                <td class="py-3 px-4 text-xs font-mono text-slate-400">${escapeHtml(ing.endpoints.join(', ') || '-')}</td>
                <td class="py-3 px-4 text-xs text-slate-400">${escapeHtml(ing.age)}</td>
            </tr>
        `).join('');
    } catch (err) {
        tbody.innerHTML = `<tr><td colspan="5" class="text-center py-8 text-rose-400">Error: ${err.message}</td></tr>`;
    }
}

// ----------------- STORAGE & CONFIG -----------------
function switchStorageSubTab(subTab) {
    currentStorageSubTab = subTab;
    document.querySelectorAll('.storage-subtab-btn').forEach(btn => {
        btn.classList.remove('border-indigo-500', 'text-indigo-400');
        btn.classList.add('border-transparent', 'text-slate-400');
    });
    document.getElementById(`storageSubTabBtn-${subTab}`)?.classList.add('border-indigo-500', 'text-indigo-400');

    document.querySelectorAll('.storage-subtab-content').forEach(el => el.classList.add('hidden'));
    document.getElementById(`storageContent-${subTab}`)?.classList.remove('hidden');

    loadStorage();
}

function loadStorage(isBackground = false) {
    switch (currentStorageSubTab) {
        case 'configmaps':
            loadConfigMaps(isBackground);
            break;
        case 'secrets':
            loadSecrets(isBackground);
            break;
        case 'pv':
            loadPV(isBackground);
            break;
        case 'pvc':
            loadPVC(isBackground);
            break;
    }
}

async function loadConfigMaps(isBackground = false) {
    const tbody = document.getElementById('configmapsTableBody');
    if (!isBackground) tbody.innerHTML = `<tr><td colspan="4" class="text-center py-8 text-slate-400">Loading configmaps...</td></tr>`;

    try {
        const res = await fetch(`/api/configmaps?namespace=${encodeURIComponent(currentNamespace)}`);
        const list = await res.json();
        if (list.length === 0) {
            tbody.innerHTML = `<tr><td colspan="4" class="text-center py-8 text-slate-400">No configmaps found</td></tr>`;
            return;
        }
        tbody.innerHTML = list.map(c => `
            <tr class="border-b border-slate-800/60 hover:bg-slate-800/30 transition search-row">
                <td class="py-3 px-4 font-semibold text-slate-200">${escapeHtml(c.name)}</td>
                <td class="py-3 px-4 text-xs text-slate-400">${escapeHtml(c.namespace)}</td>
                <td class="py-3 px-4 text-xs font-mono text-slate-300">${c.keysCount} keys (${escapeHtml(c.keys.slice(0, 4).join(', '))}${c.keys.length > 4 ? '...' : ''})</td>
                <td class="py-3 px-4 text-xs text-slate-400">${escapeHtml(c.age)}</td>
            </tr>
        `).join('');
    } catch (err) {
        tbody.innerHTML = `<tr><td colspan="4" class="text-center py-8 text-rose-400">Error: ${err.message}</td></tr>`;
    }
}

async function loadSecrets(isBackground = false) {
    const tbody = document.getElementById('secretsTableBody');
    if (!isBackground) tbody.innerHTML = `<tr><td colspan="5" class="text-center py-8 text-slate-400">Loading secrets...</td></tr>`;

    try {
        const res = await fetch(`/api/secrets?namespace=${encodeURIComponent(currentNamespace)}`);
        const list = await res.json();
        if (list.length === 0) {
            tbody.innerHTML = `<tr><td colspan="5" class="text-center py-8 text-slate-400">No secrets found</td></tr>`;
            return;
        }
        tbody.innerHTML = list.map(s => `
            <tr class="border-b border-slate-800/60 hover:bg-slate-800/30 transition search-row">
                <td class="py-3 px-4 font-semibold text-slate-200">${escapeHtml(s.name)}</td>
                <td class="py-3 px-4 text-xs text-slate-400">${escapeHtml(s.namespace)}</td>
                <td class="py-3 px-4 text-xs font-mono text-amber-400">${escapeHtml(s.type)}</td>
                <td class="py-3 px-4 text-xs font-mono text-slate-300">${s.keysCount} keys</td>
                <td class="py-3 px-4 text-xs text-slate-400">${escapeHtml(s.age)}</td>
            </tr>
        `).join('');
    } catch (err) {
        tbody.innerHTML = `<tr><td colspan="5" class="text-center py-8 text-rose-400">Error: ${err.message}</td></tr>`;
    }
}

async function loadPV(isBackground = false) {
    const tbody = document.getElementById('pvTableBody');
    if (!isBackground) tbody.innerHTML = `<tr><td colspan="6" class="text-center py-8 text-slate-400">Loading PVs...</td></tr>`;

    try {
        const res = await fetch('/api/storage/pv');
        const list = await res.json();
        if (list.length === 0) {
            tbody.innerHTML = `<tr><td colspan="6" class="text-center py-8 text-slate-400">No persistent volumes found</td></tr>`;
            return;
        }
        tbody.innerHTML = list.map(pv => `
            <tr class="border-b border-slate-800/60 hover:bg-slate-800/30 transition search-row">
                <td class="py-3 px-4 font-semibold text-slate-200">${escapeHtml(pv.name)}</td>
                <td class="py-3 px-4 text-xs font-mono text-slate-300">${escapeHtml(pv.capacity)}</td>
                <td class="py-3 px-4 text-xs font-mono text-slate-400">${escapeHtml(pv.accessModes.join(', '))}</td>
                <td class="py-3 px-4"><span class="px-2 py-0.5 rounded text-xs badge-ready">${escapeHtml(pv.status)}</span></td>
                <td class="py-3 px-4 text-xs text-slate-400">${escapeHtml(pv.storageClass || '-')}</td>
                <td class="py-3 px-4 text-xs text-slate-400">${escapeHtml(pv.age)}</td>
            </tr>
        `).join('');
    } catch (err) {
        tbody.innerHTML = `<tr><td colspan="6" class="text-center py-8 text-rose-400">Error: ${err.message}</td></tr>`;
    }
}

async function loadPVC(isBackground = false) {
    const tbody = document.getElementById('pvcTableBody');
    if (!isBackground) tbody.innerHTML = `<tr><td colspan="6" class="text-center py-8 text-slate-400">Loading PVCs...</td></tr>`;

    try {
        const res = await fetch(`/api/storage/pvc?namespace=${encodeURIComponent(currentNamespace)}`);
        const list = await res.json();
        if (list.length === 0) {
            tbody.innerHTML = `<tr><td colspan="6" class="text-center py-8 text-slate-400">No persistent volume claims found</td></tr>`;
            return;
        }
        tbody.innerHTML = list.map(pvc => `
            <tr class="border-b border-slate-800/60 hover:bg-slate-800/30 transition search-row">
                <td class="py-3 px-4 font-semibold text-slate-200">${escapeHtml(pvc.name)}</td>
                <td class="py-3 px-4 text-xs text-slate-400">${escapeHtml(pvc.namespace)}</td>
                <td class="py-3 px-4"><span class="px-2 py-0.5 rounded text-xs badge-ready">${escapeHtml(pvc.status)}</span></td>
                <td class="py-3 px-4 text-xs font-mono text-slate-300">${escapeHtml(pvc.volume || '-')}</td>
                <td class="py-3 px-4 text-xs font-mono text-slate-400">${escapeHtml(pvc.capacity || '-')}</td>
                <td class="py-3 px-4 text-xs text-slate-400">${escapeHtml(pvc.age)}</td>
            </tr>
        `).join('');
    } catch (err) {
        tbody.innerHTML = `<tr><td colspan="6" class="text-center py-8 text-rose-400">Error: ${err.message}</td></tr>`;
    }
}

// ----------------- EVENTS (AUDIT) -----------------
async function loadEvents(isBackground = false) {
    const tbody = document.getElementById('eventsTableBody');
    if (!isBackground) tbody.innerHTML = `<tr><td colspan="6" class="text-center py-8 text-slate-400">Loading events...</td></tr>`;

    try {
        const url = `/api/events?namespace=${encodeURIComponent(currentNamespace)}`;
        const res = await fetch(url);
        const events = await res.json();

        if (events.length === 0) {
            tbody.innerHTML = `<tr><td colspan="6" class="text-center py-8 text-slate-400">No events found</td></tr>`;
            return;
        }

        tbody.innerHTML = events.map(ev => {
            const isWarning = ev.type === 'Warning';
            const badgeClass = isWarning ? 'badge-warning font-semibold' : 'badge-neutral';

            return `
                <tr class="border-b border-slate-800/60 hover:bg-slate-800/30 transition search-row">
                    <td class="py-3 px-4">
                        <span class="px-2 py-0.5 rounded text-xs ${badgeClass}">
                            ${escapeHtml(ev.type)}
                        </span>
                    </td>
                    <td class="py-3 px-4 text-xs font-semibold ${isWarning ? 'text-amber-400' : 'text-slate-300'}">
                        ${escapeHtml(ev.reason)}
                    </td>
                    <td class="py-3 px-4 text-xs font-mono text-indigo-400">${escapeHtml(ev.object)}</td>
                    <td class="py-3 px-4 text-xs text-slate-300 max-w-md break-words">${escapeHtml(ev.message)}</td>
                    <td class="py-3 px-4 text-xs text-slate-400">${escapeHtml(ev.namespace || '-')}</td>
                    <td class="py-3 px-4 text-xs text-slate-400">${escapeHtml(ev.age)}</td>
                </tr>
            `;
        }).join('');
    } catch (err) {
        tbody.innerHTML = `<tr><td colspan="6" class="text-center py-8 text-rose-400">Error: ${err.message}</td></tr>`;
    }
}

// ----------------- ACTIONS (PODS & DEPLOYMENTS) -----------------
async function deletePod(namespace, name) {
    if (!confirm(`Hapus Pod ${namespace}/${name}? (Jika dikelola Deployment, pod baru akan dibuat ulang otomatis)`)) return;

    try {
        const res = await fetch(`/api/pods/delete?namespace=${encodeURIComponent(namespace)}&name=${encodeURIComponent(name)}`, {
            method: 'POST'
        });
        const data = await res.json();
        if (!res.ok) throw new Error(data.error || 'Gagal menghapus pod');
        showToast(data.message || 'Pod berhasil dihapus', 'success');
        loadPods();
    } catch (err) {
        showToast(err.message, 'error');
    }
}

function openScaleModal(namespace, name, currentReplicas) {
    document.getElementById('scaleNamespace').value = namespace;
    document.getElementById('scaleName').value = name;
    document.getElementById('scaleDeploymentTitle').textContent = `Scale ${namespace}/${name}`;
    document.getElementById('scaleReplicasInput').value = currentReplicas;
    document.getElementById('scaleModal').classList.remove('hidden');
}

function closeScaleModal() {
    document.getElementById('scaleModal').classList.add('hidden');
}

function adjustScaleCount(delta) {
    const input = document.getElementById('scaleReplicasInput');
    let val = parseInt(input.value, 10) || 0;
    val = Math.max(0, val + delta);
    input.value = val;
}

async function submitScaleDeployment() {
    const namespace = document.getElementById('scaleNamespace').value;
    const name = document.getElementById('scaleName').value;
    const replicas = parseInt(document.getElementById('scaleReplicasInput').value, 10);

    try {
        const res = await fetch('/api/deployments/scale', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ namespace, name, replicas })
        });
        const data = await res.json();
        if (!res.ok) throw new Error(data.error || 'Gagal scale deployment');
        showToast(data.message, 'success');
        closeScaleModal();
        loadDeployments();
    } catch (err) {
        showToast(err.message, 'error');
    }
}

async function restartDeployment(namespace, name) {
    if (!confirm(`Restart rollout untuk deployment ${namespace}/${name}?`)) return;

    try {
        const res = await fetch('/api/deployments/restart', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ namespace, name })
        });
        const data = await res.json();
        if (!res.ok) throw new Error(data.error || 'Gagal restart');
        showToast(data.message, 'success');
        loadDeployments();
    } catch (err) {
        showToast(err.message, 'error');
    }
}

// ----------------- POD LOGS VIEWER -----------------
function openLogsModal(namespace, name, containersJson) {
    activeLogNamespace = namespace;
    activeLogPod = name;

    const containers = JSON.parse(decodeURIComponent(containersJson));
    const containerSelect = document.getElementById('logContainerSelect');
    containerSelect.innerHTML = '';
    containers.forEach(c => {
        const opt = document.createElement('option');
        opt.value = c;
        opt.textContent = c;
        containerSelect.appendChild(opt);
    });

    activeLogContainer = containers[0] || '';
    document.getElementById('logPodTitle').textContent = `Logs: ${namespace}/${name}`;
    document.getElementById('logsModal').classList.remove('hidden');

    fetchLogs();

    // Setup auto-refresh for logs every 3 seconds while modal is open
    if (currentLogInterval) clearInterval(currentLogInterval);
    currentLogInterval = setInterval(fetchLogs, 3000);
}

function closeLogsModal() {
    document.getElementById('logsModal').classList.add('hidden');
    if (currentLogInterval) {
        clearInterval(currentLogInterval);
        currentLogInterval = null;
    }
}

async function fetchLogs() {
    if (!activeLogPod) return;
    const tail = document.getElementById('logTailSelect').value;
    const container = document.getElementById('logContainerSelect').value;

    try {
        const url = `/api/pods/logs?namespace=${encodeURIComponent(activeLogNamespace)}&name=${encodeURIComponent(activeLogPod)}&container=${encodeURIComponent(container)}&tail=${encodeURIComponent(tail)}`;
        const res = await fetch(url);
        const data = await res.json();
        const output = document.getElementById('logOutput');

        if (!res.ok) {
            output.textContent = `Error: ${data.error || 'Gagal memuat log'}`;
            return;
        }

        output.textContent = data.logs || '--- Tidak ada log output ---';
        output.scrollTop = output.scrollHeight;
    } catch (err) {
        document.getElementById('logOutput').textContent = `Koneksi gagal: ${err.message}`;
    }
}

function copyLogsToClipboard() {
    const text = document.getElementById('logOutput').textContent;
    navigator.clipboard.writeText(text).then(() => {
        showToast('Log berhasil disalin ke clipboard', 'success');
    });
}

// ----------------- SETTINGS & CLUSTER CONNECTION -----------------
async function loadSettingsInfo() {
    try {
        const res = await fetch('/api/status');
        const data = await res.json();
        document.getElementById('settingConnected').textContent = data.connected ? 'Terhubung (Online)' : 'Tidak Terhubung (Offline)';
        document.getElementById('settingConnectedVia').textContent = data.connectedVia || '-';
        document.getElementById('settingAPIServer').textContent = data.serverURL || '-';
        document.getElementById('settingVersion').textContent = data.version || '-';
        document.getElementById('settingError').textContent = data.error || 'None';
    } catch (err) {
        console.error('Error loading settings info', err);
    }
}

async function connectCluster(mode) {
    let payload = { mode };

    if (mode === 'kubeconfig') {
        const content = document.getElementById('kubeconfigInput').value.trim();
        if (!content) {
            showToast('Silakan tempel isi file kubeconfig', 'error');
            return;
        }
        payload.kubeconfigContent = content;
    } else if (mode === 'token') {
        const server = document.getElementById('tokenServerInput').value.trim();
        const token = document.getElementById('tokenInput').value.trim();
        const insecure = document.getElementById('tokenInsecureCheck').checked;
        if (!server || !token) {
            showToast('Server URL dan Token wajib diisi', 'error');
            return;
        }
        payload.server = server;
        payload.token = token;
        payload.insecure = insecure;
    }

    try {
        showToast('Menghubungkan ke Kubernetes...', 'info');
        const res = await fetch('/api/connect', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
        });
        const data = await res.json();
        if (!res.ok) throw new Error(data.error || 'Koneksi gagal');

        showToast('Berhasil terhubung ke cluster Kubernetes!', 'success');
        checkClusterStatus();
        loadNamespaces();
        loadSettingsInfo();
        if (currentTab === 'overview') loadOverview();
    } catch (err) {
        showToast(err.message, 'error');
        checkClusterStatus();
    }
}

// ----------------- UTILITIES -----------------
function filterTable(query) {
    document.querySelectorAll('.search-row').forEach(row => {
        const text = row.textContent.toLowerCase();
        if (text.includes(query)) {
            row.style.display = '';
        } else {
            row.style.display = 'none';
        }
    });
}

function showToast(message, type = 'info') {
    const toast = document.getElementById('toast');
    if (!toast) return;

    let colors = 'bg-slate-800 text-slate-200 border-slate-700';
    if (type === 'success') colors = 'bg-emerald-950 text-emerald-200 border-emerald-800';
    else if (type === 'error') colors = 'bg-rose-950 text-rose-200 border-rose-800';

    toast.className = `fixed bottom-6 right-6 px-4 py-3 rounded-lg border shadow-2xl transition duration-300 z-50 text-sm font-medium ${colors}`;
    toast.textContent = message;
    toast.classList.remove('hidden');

    setTimeout(() => {
        toast.classList.add('hidden');
    }, 4000);
}

function escapeHtml(str) {
    if (!str) return '';
    return String(str)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#039;');
}

// ----------------- AI COPILOT FUNCTIONS -----------------
async function loadAIConfigInfo() {
    try {
        const res = await fetch('/api/ai/config');
        if (!res.ok) return;
        const data = await res.json();
        const badge = document.getElementById('aiProviderBadge');
        if (badge) {
            if (data.hasKey) {
                badge.className = 'text-[11px] px-2.5 py-0.5 rounded-full bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 font-medium';
                badge.textContent = `Active: ${data.provider.toUpperCase()} (${data.model})`;
            } else {
                badge.className = 'text-[11px] px-2.5 py-0.5 rounded-full bg-purple-500/10 text-purple-400 border border-purple-500/20 font-medium';
                badge.textContent = 'Active: Built-in K8s Engine';
            }
        }
    } catch (err) {
        console.error('Error loading AI config:', err);
    }
}

async function openAIConfigModal() {
    try {
        const res = await fetch('/api/ai/config');
        if (res.ok) {
            const data = await res.json();
            document.getElementById('aiProviderSelect').value = data.provider || 'gemini';
            document.getElementById('aiModelInput').value = data.model || 'gemini-2.5-flash';
            if (data.maskedKey) {
                document.getElementById('aiApiKeyInput').placeholder = `Tersimpan: ${data.maskedKey}`;
            }
        }
    } catch (e) {}
    document.getElementById('aiConfigModal').classList.remove('hidden');
}

function closeAIConfigModal() {
    document.getElementById('aiConfigModal').classList.add('hidden');
}

async function saveAIConfig() {
    const provider = document.getElementById('aiProviderSelect').value;
    const model = document.getElementById('aiModelInput').value.trim();
    const apiKey = document.getElementById('aiApiKeyInput').value.trim();

    try {
        const res = await fetch('/api/ai/config', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ provider, model, apiKey })
        });
        const data = await res.json();
        if (!res.ok) throw new Error(data.error || 'Gagal menyimpan');
        showToast(data.message, 'success');
        closeAIConfigModal();
        loadAIConfigInfo();
    } catch (err) {
        showToast(err.message, 'error');
    }
}

function useQuickPrompt(text) {
    const input = document.getElementById('aiPromptInput');
    if (!input) return;
    input.value = text;
    sendAIMessage();
}

function clearAIChat() {
    const history = document.getElementById('aiChatHistory');
    if (!history) return;
    history.innerHTML = `
        <div class="flex gap-3">
            <div class="w-8 h-8 rounded-lg bg-gradient-to-tr from-purple-600 to-indigo-500 flex items-center justify-center shrink-0 shadow-md">
                <svg class="w-4 h-4 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"></path></svg>
            </div>
            <div class="flex-1 bg-slate-900 border border-slate-800 rounded-2xl rounded-tl-none p-4 text-xs text-slate-200 leading-relaxed shadow-sm">
                <p class="font-semibold text-purple-400 mb-1">Halo! Saya K8s AI Copilot Anda 🤖</p>
                <p>Percakapan telah dibersihkan. Silakan ajukan pertanyaan atau paste log error server Anda!</p>
            </div>
        </div>
    `;
}

async function sendAIMessage() {
    const input = document.getElementById('aiPromptInput');
    const prompt = input.value.trim();
    if (!prompt) return;

    const attachContext = document.getElementById('aiAttachContext')?.checked ?? true;
    const history = document.getElementById('aiChatHistory');
    const sendBtn = document.getElementById('aiSendBtn');

    // 1. Append User Message
    const userMsgId = 'user-msg-' + Date.now();
    const userBubble = document.createElement('div');
    userBubble.id = userMsgId;
    userBubble.className = 'flex justify-end gap-3';
    userBubble.innerHTML = `
        <div class="max-w-2xl bg-gradient-to-r from-purple-600 to-indigo-600 text-white rounded-2xl rounded-tr-none p-3.5 text-xs shadow-md">
            <p class="whitespace-pre-wrap">${escapeHtml(prompt)}</p>
        </div>
    `;
    history.appendChild(userBubble);
    input.value = '';
    history.scrollTop = history.scrollHeight;

    // 2. Append Thinking Indicator
    const thinkingId = 'thinking-' + Date.now();
    const thinkingBubble = document.createElement('div');
    thinkingBubble.id = thinkingId;
    thinkingBubble.className = 'flex gap-3';
    thinkingBubble.innerHTML = `
        <div class="w-8 h-8 rounded-lg bg-gradient-to-tr from-purple-600 to-indigo-500 flex items-center justify-center shrink-0 shadow-md animate-pulse">
            <svg class="w-4 h-4 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"></path></svg>
        </div>
        <div class="bg-slate-900 border border-slate-800 rounded-2xl rounded-tl-none p-3.5 text-xs text-slate-400 flex items-center gap-2">
            <span class="w-2 h-2 rounded-full bg-purple-400 animate-ping"></span>
            <span>Menganalisis cluster & menyusun solusi...</span>
        </div>
    `;
    history.appendChild(thinkingBubble);
    history.scrollTop = history.scrollHeight;

    sendBtn.disabled = true;
    sendBtn.classList.add('opacity-50');

    try {
        const res = await fetch('/api/ai/chat', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                prompt: prompt,
                attachClusterContext: attachContext
            })
        });

        const data = await res.json();
        thinkingBubble.remove();

        if (!res.ok) throw new Error(data.error || 'Gagal memproses permintaan AI');

        // Render AI Reply with Markdown
        let renderedHtml = '';
        if (window.marked && typeof window.marked.parse === 'function') {
            renderedHtml = window.marked.parse(data.reply || '');
        } else {
            renderedHtml = `<p class="whitespace-pre-wrap">${escapeHtml(data.reply || '')}</p>`;
        }

        const aiBubble = document.createElement('div');
        aiBubble.className = 'flex gap-3';
        aiBubble.innerHTML = `
            <div class="w-8 h-8 rounded-lg bg-gradient-to-tr from-purple-600 to-indigo-500 flex items-center justify-center shrink-0 shadow-md">
                <svg class="w-4 h-4 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"></path></svg>
            </div>
            <div class="flex-1 max-w-3xl bg-slate-900 border border-slate-800 rounded-2xl rounded-tl-none p-4 text-xs text-slate-200 leading-relaxed shadow-sm markdown-body space-y-2">
                <div class="flex items-center justify-between pb-2 mb-2 border-b border-slate-800/80 text-[10px] text-slate-400">
                    <span class="font-semibold text-purple-400">AI Response</span>
                    <span class="font-mono">${escapeHtml(data.provider || 'AI Engine')}</span>
                </div>
                <div class="ai-content-text">${renderedHtml}</div>
            </div>
        `;

        // Attach copy buttons to any pre/code blocks
        aiBubble.querySelectorAll('pre').forEach(pre => {
            pre.className = 'relative bg-slate-950 border border-slate-800 rounded-lg p-3 my-2 text-xs font-mono text-slate-200 overflow-x-auto';
            const copyBtn = document.createElement('button');
            copyBtn.className = 'absolute top-2 right-2 px-2 py-0.5 bg-slate-800 hover:bg-slate-700 text-[10px] text-slate-300 rounded border border-slate-700 transition';
            copyBtn.textContent = 'Copy';
            copyBtn.onclick = () => {
                const codeText = pre.querySelector('code')?.innerText || pre.innerText;
                navigator.clipboard.writeText(codeText.replace(/^Copy\n/, '')).then(() => {
                    copyBtn.textContent = 'Copied!';
                    setTimeout(() => copyBtn.textContent = 'Copy', 2000);
                });
            };
            pre.appendChild(copyBtn);
        });

        history.appendChild(aiBubble);
        history.scrollTop = history.scrollHeight;
    } catch (err) {
        thinkingBubble.remove();
        const errBubble = document.createElement('div');
        errBubble.className = 'flex gap-3';
        errBubble.innerHTML = `
            <div class="w-8 h-8 rounded-lg bg-rose-600 flex items-center justify-center shrink-0">
                <svg class="w-4 h-4 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path></svg>
            </div>
            <div class="bg-rose-950/60 border border-rose-800 text-rose-200 rounded-2xl rounded-tl-none p-3.5 text-xs">
                ${escapeHtml(err.message)}
            </div>
        `;
        history.appendChild(errBubble);
    } finally {
        sendBtn.disabled = false;
        sendBtn.classList.remove('opacity-50');
    }
}


