// 管理后台 JavaScript

const API_BASE = '/api/v1/admin';
const ITEMS_PER_PAGE = 10;

// 初始化
document.addEventListener('DOMContentLoaded', function() {
    initializeNavigation();
    initializeModal();
    updateTime();
    loadDashboard();
    
    setInterval(updateTime, 1000);
});

// 更新时间
function updateTime() {
    const now = new Date();
    document.getElementById('currentTime').textContent = now.toLocaleTimeString('zh-CN');
}

// 初始化导航
function initializeNavigation() {
    const navLinks = document.querySelectorAll('.nav-link');
    navLinks.forEach(link => {
        link.addEventListener('click', function(e) {
            e.preventDefault();
            const section = this.dataset.section;
            switchSection(section);
            
            // 更新活跃导航项
            navLinks.forEach(l => l.classList.remove('active'));
            this.classList.add('active');
        });
    });
}

// 切换页面
function switchSection(section) {
    const sections = document.querySelectorAll('.section');
    sections.forEach(s => s.classList.remove('active'));
    
    const targetSection = document.getElementById(section + '-section');
    if (targetSection) {
        targetSection.classList.add('active');
        
        // 加载对应的数据
        switch(section) {
            case 'agents':
                loadAgents();
                break;
            case 'files':
                loadFiles();
                break;
            case 'configs':
                loadStorageConfigs();
                loadStorageTypes();
                break;
            case 'reports':
                loadReports();
                break;
            case 'dashboard':
                loadDashboard();
                break;
        }
    }
}

// 加载仪表板
async function loadDashboard() {
    try {
        const statsResponse = await fetch(`${API_BASE}/reports/stats`);
        const stats = await statsResponse.json();
        
        if (stats.code === 200) {
            const data = stats.data || {};
            document.getElementById('totalAgents').textContent = data.total_agents || 0;
            document.getElementById('totalFiles').textContent = data.total_files || 0;
            document.getElementById('totalErrors').textContent = data.total_errors || 0;
        }
        
        // 计算在线 Agent
        const agentsResponse = await fetch(`${API_BASE}/agents`);
        const agents = await agentsResponse.json();
        if (agents.code === 200 && agents.data && agents.data.records) {
            const now = Date.now();
            const onlineCount = agents.data.records.filter(agent => {
                const lastHeartbeat = new Date(agent.last_heartbeat).getTime();
                return (now - lastHeartbeat) < 5 * 60 * 1000; // 5分钟内
            }).length;
            document.getElementById('onlineAgents').textContent = onlineCount;
        }
    } catch (error) {
        console.error('加载仪表板数据失败:', error);
    }
}

// 加载 Agent 列表
async function loadAgents(page = 1) {
    try {
        const response = await fetch(`${API_BASE}/agents?page=${page}&per_page=${ITEMS_PER_PAGE}`);
        const data = await response.json();
        
        const tbody = document.getElementById('agentsTableBody');
        tbody.innerHTML = '';
        
        if (data.code !== 200 || !data.data || !data.data.records) {
            tbody.innerHTML = '<tr><td colspan="8" class="loading">无数据</td></tr>';
            return;
        }
        
        data.data.records.forEach(agent => {
            const lastHeartbeat = new Date(agent.last_heartbeat);
            const now = Date.now();
            const isOnline = (now - lastHeartbeat.getTime()) < 5 * 60 * 1000;
            
            const row = document.createElement('tr');
            row.innerHTML = `
                <td><code>${agent.agent_id}</code></td>
                <td>${agent.email_prefix || '-'}</td>
                <td>${agent.hostname || '-'}</td>
                <td>${agent.ip_address || '-'}</td>
                <td>${agent.os_version || '-'}</td>
                <td><span class="status-badge-table ${isOnline ? 'status-online' : 'status-offline'}">
                    ${isOnline ? '在线' : '离线'}
                </span></td>
                <td>${formatTime(lastHeartbeat)}</td>
                <td>
                    <button class="btn btn-small" onclick="viewAgentDetails('${agent.agent_id}')">详情</button>
                </td>
            `;
            tbody.appendChild(row);
        });
    } catch (error) {
        console.error('加载 Agent 失败:', error);
        const tbody = document.getElementById('agentsTableBody');
        tbody.innerHTML = '<tr><td colspan="8" class="loading">加载失败</td></tr>';
    }
}

// 加载文件列表
async function loadFiles(page = 1) {
    try {
        const response = await fetch(`${API_BASE}/files?page=${page}&per_page=${ITEMS_PER_PAGE}`);
        const data = await response.json();
        
        const tbody = document.getElementById('filesTableBody');
        tbody.innerHTML = '';
        
        if (data.code !== 200 || !data.data || !data.data.records) {
            tbody.innerHTML = '<tr><td colspan="8" class="loading">无数据</td></tr>';
            return;
        }
        
        data.data.records.forEach(file => {
            const uploadTime = new Date(file.upload_time);
            const row = document.createElement('tr');
            row.innerHTML = `
                <td><code>${file.file_id}</code></td>
                <td>${file.file_name || '-'}</td>
                <td><code>${file.agent_id || '-'}</code></td>
                <td>${file.email_prefix || '-'}</td>
                <td>${formatFileSize(file.file_size || 0)}</td>
                <td>${formatTime(uploadTime)}</td>
                <td><span class="status-badge-table status-success">已上传</span></td>
                <td>
                    <button class="btn btn-small" onclick="viewFileDetails('${file.file_id}')">详情</button>
                </td>
            `;
            tbody.appendChild(row);
        });
    } catch (error) {
        console.error('加载文件失败:', error);
        const tbody = document.getElementById('filesTableBody');
        tbody.innerHTML = '<tr><td colspan="8" class="loading">加载失败</td></tr>';
    }
}

// 加载配置列表
async function loadConfigs(page = 1) {
    try {
        const response = await fetch(`${API_BASE}/configs?page=${page}&per_page=${ITEMS_PER_PAGE}`);
        const data = await response.json();
        
        const tbody = document.getElementById('configsTableBody');
        tbody.innerHTML = '';
        
        if (data.code !== 200 || !data.data || !data.data.records) {
            tbody.innerHTML = '<tr><td colspan="6" class="loading">无数据</td></tr>';
            return;
        }
        
        data.data.records.forEach(config => {
            const createTime = new Date(config.created_at);
            const updateTime = new Date(config.updated_at);
            const row = document.createElement('tr');
            row.innerHTML = `
                <td><code>${config.config_id}</code></td>
                <td>${config.name || '-'}</td>
                <td>${config.description || '-'}</td>
                <td>${formatTime(createTime)}</td>
                <td>${formatTime(updateTime)}</td>
                <td>
                    <button class="btn btn-small" onclick="editConfig('${config.config_id}')">编辑</button>
                    <button class="btn btn-small btn-danger" onclick="deleteConfig('${config.config_id}')">删除</button>
                </td>
            `;
            tbody.appendChild(row);
        });
    } catch (error) {
        console.error('加载配置失败:', error);
        const tbody = document.getElementById('configsTableBody');
        tbody.innerHTML = '<tr><td colspan="6" class="loading">加载失败</td></tr>';
    }
}

// 加载报表
async function loadReports() {
    try {
        // 系统统计
        const statsResponse = await fetch(`${API_BASE}/reports/stats`);
        const stats = await statsResponse.json();
        if (stats.code === 200) {
            const data = stats.data || {};
            document.getElementById('statsReport').innerHTML = `
                <div><span class="report-label">总 Agent 数:</span> <span class="report-value">${data.total_agents || 0}</span></div>
                <div><span class="report-label">总文件数:</span> <span class="report-value">${data.total_files || 0}</span></div>
                <div><span class="report-label">错误数:</span> <span class="report-value">${data.total_errors || 0}</span></div>
            `;
        }
        
        // Agent 统计
        const agentStatsResponse = await fetch(`${API_BASE}/reports/agents`);
        const agentStats = await agentStatsResponse.json();
        if (agentStats.code === 200) {
            const data = agentStats.data || {};
            document.getElementById('agentStatsReport').innerHTML = `
                <div><span class="report-label">平均扫描文件数:</span> <span class="report-value">${data.avg_scan_count || 0}</span></div>
                <div><span class="report-label">平均上传文件数:</span> <span class="report-value">${data.avg_upload_count || 0}</span></div>
                <div><span class="report-label">平均错误数:</span> <span class="report-value">${data.avg_error_count || 0}</span></div>
            `;
        }
        
        // 文件统计
        const fileStatsResponse = await fetch(`${API_BASE}/reports/files`);
        const fileStats = await fileStatsResponse.json();
        if (fileStats.code === 200) {
            const data = fileStats.data || {};
            document.getElementById('fileStatsReport').innerHTML = `
                <div><span class="report-label">总文件大小:</span> <span class="report-value">${formatFileSize(data.total_size || 0)}</span></div>
                <div><span class="report-label">平均文件大小:</span> <span class="report-value">${formatFileSize(data.avg_size || 0)}</span></div>
                <div><span class="report-label">最大文件大小:</span> <span class="report-value">${formatFileSize(data.max_size || 0)}</span></div>
            `;
        }
    } catch (error) {
        console.error('加载报表失败:', error);
    }
}

// 刷新函数
function refreshAgents() {
    loadAgents();
}

function refreshFiles() {
    loadFiles();
}

function refreshConfigs() {
    loadConfigs();
}

// 查看详情
function viewAgentDetails(agentId) {
    openModal(`
        <h3>Agent 详情: ${agentId}</h3>
        <div id="agentDetailContent" class="loading">加载中...</div>
    `);
    
    fetch(`${API_BASE}/agents/${agentId}`)
        .then(r => r.json())
        .then(data => {
            if (data.code === 200 && data.data) {
                const agent = data.data;
                document.getElementById('agentDetailContent').innerHTML = `
                    <table style="width: 100%; border-collapse: collapse;">
                        <tr><td style="padding: 8px; border-bottom: 1px solid #ddd;"><strong>Agent ID:</strong></td><td style="padding: 8px; border-bottom: 1px solid #ddd;"><code>${agent.agent_id}</code></td></tr>
                        <tr><td style="padding: 8px; border-bottom: 1px solid #ddd;"><strong>邮箱前缀:</strong></td><td style="padding: 8px; border-bottom: 1px solid #ddd;">${agent.email_prefix || '-'}</td></tr>
                        <tr><td style="padding: 8px; border-bottom: 1px solid #ddd;"><strong>主机名:</strong></td><td style="padding: 8px; border-bottom: 1px solid #ddd;">${agent.hostname || '-'}</td></tr>
                        <tr><td style="padding: 8px; border-bottom: 1px solid #ddd;"><strong>IP 地址:</strong></td><td style="padding: 8px; border-bottom: 1px solid #ddd;">${agent.ip_address || '-'}</td></tr>
                        <tr><td style="padding: 8px; border-bottom: 1px solid #ddd;"><strong>OS 版本:</strong></td><td style="padding: 8px; border-bottom: 1px solid #ddd;">${agent.os_version || '-'}</td></tr>
                        <tr><td style="padding: 8px; border-bottom: 1px solid #ddd;"><strong>最后心跳:</strong></td><td style="padding: 8px; border-bottom: 1px solid #ddd;">${formatTime(new Date(agent.last_heartbeat))}</td></tr>
                        <tr><td style="padding: 8px;"><strong>创建时间:</strong></td><td style="padding: 8px;">${formatTime(new Date(agent.created_at))}</td></tr>
                    </table>
                `;
            }
        })
        .catch(err => {
            document.getElementById('agentDetailContent').innerHTML = '加载失败: ' + err.message;
        });
}

function viewFileDetails(fileId) {
    openModal(`
        <h3>文件详情: ${fileId}</h3>
        <div id="fileDetailContent" class="loading">加载中...</div>
    `);
    
    fetch(`${API_BASE}/files/${fileId}`)
        .then(r => r.json())
        .then(data => {
            if (data.code === 200 && data.data) {
                const file = data.data;
                document.getElementById('fileDetailContent').innerHTML = `
                    <table style="width: 100%; border-collapse: collapse;">
                        <tr><td style="padding: 8px; border-bottom: 1px solid #ddd;"><strong>文件 ID:</strong></td><td style="padding: 8px; border-bottom: 1px solid #ddd;"><code>${file.file_id}</code></td></tr>
                        <tr><td style="padding: 8px; border-bottom: 1px solid #ddd;"><strong>文件名:</strong></td><td style="padding: 8px; border-bottom: 1px solid #ddd;">${file.file_name || '-'}</td></tr>
                        <tr><td style="padding: 8px; border-bottom: 1px solid #ddd;"><strong>Agent ID:</strong></td><td style="padding: 8px; border-bottom: 1px solid #ddd;"><code>${file.agent_id || '-'}</code></td></tr>
                        <tr><td style="padding: 8px; border-bottom: 1px solid #ddd;"><strong>邮箱前缀:</strong></td><td style="padding: 8px; border-bottom: 1px solid #ddd;">${file.email_prefix || '-'}</td></tr>
                        <tr><td style="padding: 8px; border-bottom: 1px solid #ddd;"><strong>文件大小:</strong></td><td style="padding: 8px; border-bottom: 1px solid #ddd;">${formatFileSize(file.file_size || 0)}</td></tr>
                        <tr><td style="padding: 8px;"><strong>上传时间:</strong></td><td style="padding: 8px;">${formatTime(new Date(file.upload_time))}</td></tr>
                    </table>
                `;
            }
        })
        .catch(err => {
            document.getElementById('fileDetailContent').innerHTML = '加载失败: ' + err.message;
        });
}

// 配置管理
function openAddConfig() {
    openModal(`
        <h3>新增配置</h3>
        <form id="configForm">
            <div style="margin-bottom: 15px;">
                <label>配置名称:</label>
                <input type="text" id="configName" style="width: 100%; padding: 8px; margin-top: 5px; border: 1px solid #ddd; border-radius: 4px;">
            </div>
            <div style="margin-bottom: 15px;">
                <label>描述:</label>
                <textarea id="configDesc" style="width: 100%; padding: 8px; margin-top: 5px; border: 1px solid #ddd; border-radius: 4px; min-height: 100px;"></textarea>
            </div>
            <button type="button" class="btn btn-primary" onclick="saveConfig()">保存</button>
        </form>
    `);
}

function editConfig(configId) {
    alert('编辑功能开发中...');
}

function deleteConfig(configId) {
    if (confirm('确认删除此配置？')) {
        fetch(`${API_BASE}/configs/${configId}`, { method: 'DELETE' })
            .then(r => r.json())
            .then(data => {
                if (data.code === 200) {
                    alert('删除成功');
                    loadConfigs();
                } else {
                    alert('删除失败');
                }
            });
    }
}

function saveConfig() {
    const name = document.getElementById('configName').value;
    const description = document.getElementById('configDesc').value;
    
    if (!name) {
        alert('请输入配置名称');
        return;
    }
    
    fetch(`${API_BASE}/configs`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name, description })
    })
    .then(r => r.json())
    .then(data => {
        if (data.code === 200) {
            alert('保存成功');
            closeModal();
            loadConfigs();
        } else {
            alert('保存失败');
        }
    });
}

// 模态框
function initializeModal() {
    const modal = document.getElementById('modal');
    const closeBtn = document.querySelector('.close');
    
    closeBtn.addEventListener('click', closeModal);
    window.addEventListener('click', function(e) {
        if (e.target === modal) closeModal();
    });
}

function openModal(content) {
    document.getElementById('modalBody').innerHTML = content;
    document.getElementById('modal').style.display = 'block';
}

function closeModal() {
    document.getElementById('modal').style.display = 'none';
}

// 工具函数
function formatTime(date) {
    if (!date) return '-';
    return date.toLocaleString('zh-CN');
}

function formatFileSize(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

// 修改密码功能
function openChangePasswordModal() {
    document.getElementById('changePasswordModal').style.display = 'block';
    document.getElementById('changePasswordForm').reset();
    document.getElementById('passwordError').style.display = 'none';
}

function closeChangePasswordModal() {
    document.getElementById('changePasswordModal').style.display = 'none';
    document.getElementById('changePasswordForm').reset();
    document.getElementById('passwordError').style.display = 'none';
}

// 修改密码表单提交
document.addEventListener('DOMContentLoaded', function() {
    const form = document.getElementById('changePasswordForm');
    if (form) {
        form.addEventListener('submit', async function(e) {
            e.preventDefault();

            const oldPassword = document.getElementById('oldPassword').value;
            const newPassword = document.getElementById('newPassword').value;
            const confirmPassword = document.getElementById('confirmPassword').value;
            const errorDiv = document.getElementById('passwordError');

            // 验证新密码
            if (newPassword.length < 6) {
                errorDiv.textContent = '新密码长度至少为6位';
                errorDiv.style.display = 'block';
                return;
            }

            // 验证密码一致性
            if (newPassword !== confirmPassword) {
                errorDiv.textContent = '两次输入的新密码不一致';
                errorDiv.style.display = 'block';
                return;
            }

            try {
                const response = await fetch('/api/v1/auth/password', {
                    method: 'PUT',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        old_password: oldPassword,
                        new_password: newPassword
                    }),
                });

                const data = await response.json();

                if (response.ok && data.success) {
                    alert('密码修改成功！即将跳转到登录页面...');
                    closeChangePasswordModal();
                    // 延迟跳转，让用户看到成功消息
                    setTimeout(() => {
                        window.location.href = '/login';
                    }, 1000);
                } else {
                    errorDiv.textContent = data.message || '密码修改失败';
                    errorDiv.style.display = 'block';
                }
            } catch (error) {
                console.error('Change password error:', error);
                errorDiv.textContent = '网络错误，请稍后重试';
                errorDiv.style.display = 'block';
            }
        });
    }
});

// 登出功能
async function handleLogout() {
    if (!confirm('确认要退出登录吗？')) {
        return;
    }

    try {
        const response = await fetch('/api/v1/auth/logout', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
        });

        if (response.ok) {
            window.location.href = '/login';
        } else {
            alert('退出登录失败');
        }
    } catch (error) {
        console.error('Logout error:', error);
        alert('网络错误，请稍后重试');
    }
}

// 点击模态框外部关闭
window.addEventListener('click', function(e) {
    const modal = document.getElementById('changePasswordModal');
    if (e.target === modal) {
        closeChangePasswordModal();
    }
});

