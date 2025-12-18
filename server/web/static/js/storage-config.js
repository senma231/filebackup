// 存储配置管理 JavaScript

const STORAGE_API_BASE = '/api/v1/admin/storage-configs';
let storageTypes = [];
let storageConfigs = [];

// 加载存储配置
function loadStorageConfigs() {
    fetch(STORAGE_API_BASE)
        .then(r => r.json())
        .then(data => {
            if (data.code === 200) {
                storageConfigs = data.data || [];
                renderStorageConfigsTable();
            } else {
                console.error('Failed to load storage configs:', data.message);
                document.getElementById('storageConfigsTableBody').innerHTML =
                    '<tr><td colspan="8" style="text-align: center; color: #999;">加载失败</td></tr>';
            }
        })
        .catch(err => {
            console.error('Error loading storage configs:', err);
            document.getElementById('storageConfigsTableBody').innerHTML =
                '<tr><td colspan="8" style="text-align: center; color: #999;">加载失败</td></tr>';
        });
}

// 加载存储类型
function loadStorageTypes() {
    fetch(STORAGE_API_BASE + '/types')
        .then(r => r.json())
        .then(data => {
            if (data.code === 200) {
                storageTypes = data.data || [];
                // 更新筛选下拉框
                const filterSelect = document.getElementById('storageTypeFilter');
                if (filterSelect) {
                    storageTypes.forEach(type => {
                        const option = document.createElement('option');
                        option.value = type.value;
                        option.textContent = type.label;
                        filterSelect.appendChild(option);
                    });
                }
                // 更新表单下拉框
                updateStorageTypeSelect();
            }
        })
        .catch(err => console.error('Error loading storage types:', err));
}

// 更新存储类型下拉框
function updateStorageTypeSelect() {
    const select = document.getElementById('storageType');
    if (!select) return;

    // 清空现有选项（保留第一个提示选项）
    select.innerHTML = '<option value="">请选择存储类型</option>';

    storageTypes.forEach(type => {
        const option = document.createElement('option');
        option.value = type.value;
        option.textContent = `${type.label} - ${type.description}`;
        select.appendChild(option);
    });
}

// 渲染存储配置表格
function renderStorageConfigsTable() {
    const tbody = document.getElementById('storageConfigsTableBody');
    if (!tbody) return;

    if (!storageConfigs || storageConfigs.length === 0) {
        tbody.innerHTML = '<tr><td colspan="8" style="text-align: center; color: #999;">暂无配置</td></tr>';
        return;
    }

    tbody.innerHTML = storageConfigs.map(config => `
        <tr>
            <td>${config.id}</td>
            <td>${escapeHtml(config.name)}</td>
            <td><span class="storage-type-badge">${getStorageTypeLabel(config.storage_type)}</span></td>
            <td>${config.target_agent_id ? escapeHtml(config.target_agent_id) : '<span style="color: #999;">全局</span>'}</td>
            <td>${config.priority}</td>
            <td>
                <span class="${config.is_active ? 'status-active' : 'status-inactive'}">
                    ${config.is_active ? '启用' : '禁用'}
                </span>
            </td>
            <td>${formatTime(new Date(config.created_at))}</td>
            <td>
                <div class="action-buttons">
                    <button class="action-btn action-btn-edit" onclick="editStorageConfig(${config.id})">编辑</button>
                    <button class="action-btn action-btn-toggle" onclick="toggleStorageConfig(${config.id}, ${!config.is_active})">
                        ${config.is_active ? '禁用' : '启用'}
                    </button>
                    <button class="action-btn action-btn-delete" onclick="deleteStorageConfig(${config.id})">删除</button>
                </div>
            </td>
        </tr>
    `).join('');
}

// 获取存储类型标签
function getStorageTypeLabel(value) {
    const type = storageTypes.find(t => t.value === value);
    return type ? type.label : value;
}

// 筛选存储配置
function filterStorageConfigs() {
    const filterValue = document.getElementById('storageTypeFilter').value;
    if (!filterValue) {
        loadStorageConfigs();
        return;
    }

    fetch(`${STORAGE_API_BASE}?storage_type=${filterValue}`)
        .then(r => r.json())
        .then(data => {
            if (data.code === 200) {
                storageConfigs = data.data || [];
                renderStorageConfigsTable();
            }
        })
        .catch(err => console.error('Error filtering configs:', err));
}

// 刷新存储配置
function refreshStorageConfigs() {
    document.getElementById('storageTypeFilter').value = '';
    loadStorageConfigs();
}

// 打开新增配置模态框
function openAddStorageConfig() {
    document.getElementById('storageConfigModalTitle').textContent = '➕ 新增存储配置';
    document.getElementById('configId').value = '';
    document.getElementById('storageConfigForm').reset();
    document.getElementById('configActive').checked = true;
    document.getElementById('storageConfigFields').innerHTML = '<p style="color: #999;">请先选择存储类型</p>';
    document.getElementById('storageConfigError').style.display = 'none';
    document.getElementById('storageConfigModal').style.display = 'block';
}

// 关闭存储配置模态框
function closeStorageConfigModal() {
    document.getElementById('storageConfigModal').style.display = 'none';
}

// 编辑存储配置
function editStorageConfig(id) {
    const config = storageConfigs.find(c => c.id === id);
    if (!config) {
        alert('配置不存在');
        return;
    }

    document.getElementById('storageConfigModalTitle').textContent = '✏️ 编辑存储配置';
    document.getElementById('configId').value = config.id;
    document.getElementById('configName').value = config.name;
    document.getElementById('storageType').value = config.storage_type;
    document.getElementById('configDescription').value = config.description || '';
    document.getElementById('targetAgent').value = config.target_agent_id || '';
    document.getElementById('configPriority').value = config.priority;
    document.getElementById('configActive').checked = config.is_active;

    // 更新存储配置表单
    updateStorageConfigForm(config.config_data);

    document.getElementById('storageConfigModal').style.display = 'block';
}

// 删除存储配置
function deleteStorageConfig(id) {
    if (!confirm('确认要删除此存储配置吗？删除后无法恢复！')) {
        return;
    }

    fetch(`${STORAGE_API_BASE}/${id}`, {
        method: 'DELETE'
    })
    .then(r => r.json())
    .then(data => {
        if (data.code === 200) {
            alert('删除成功');
            loadStorageConfigs();
        } else {
            alert('删除失败：' + data.message);
        }
    })
    .catch(err => {
        console.error('Error deleting config:', err);
        alert('删除失败，请稍后重试');
    });
}

// 启用/禁用存储配置
function toggleStorageConfig(id, newStatus) {
    fetch(`${STORAGE_API_BASE}/${id}`, {
        method: 'PUT',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({
            is_active: newStatus
        })
    })
    .then(r => r.json())
    .then(data => {
        if (data.code === 200) {
            alert(newStatus ? '已启用' : '已禁用');
            loadStorageConfigs();
        } else {
            alert('操作失败：' + data.message);
        }
    })
    .catch(err => {
        console.error('Error toggling config:', err);
        alert('操作失败，请稍后重试');
    });
}

// 更新存储配置表单（根据类型显示不同字段）
function updateStorageConfigForm(existingData) {
    const storageType = document.getElementById('storageType').value;
    const fieldsContainer = document.getElementById('storageConfigFields');

    if (!storageType) {
        fieldsContainer.innerHTML = '<p style="color: #999;">请先选择存储类型</p>';
        return;
    }

    let configData = {};
    if (existingData) {
        try {
            configData = typeof existingData === 'string' ? JSON.parse(existingData) : existingData;
        } catch (e) {
            console.error('Failed to parse config data:', e);
        }
    }

    fieldsContainer.innerHTML = '<h3 style="font-size: 16px; margin-bottom: 15px;">存储配置参数</h3>' +
        getStorageConfigFields(storageType, configData);
}

// 获取不同存储类型的配置字段
function getStorageConfigFields(storageType, configData = {}) {
    switch (storageType) {
        case 'sftp':
            return `
                <div class="form-group">
                    <label>服务器地址 *</label>
                    <input type="text" id="field_host" value="${configData.host || ''}" required placeholder="例如：192.168.1.100">
                </div>
                <div class="form-group">
                    <label>端口 *</label>
                    <input type="number" id="field_port" value="${configData.port || 22}" required min="1" max="65535">
                </div>
                <div class="form-group">
                    <label>用户名 *</label>
                    <input type="text" id="field_username" value="${configData.username || ''}" required>
                </div>
                <div class="form-group">
                    <label>密码</label>
                    <input type="password" id="field_password" value="${configData.password || ''}" placeholder="留空则使用SSH密钥">
                </div>
                <div class="form-group">
                    <label>SSH私钥路径</label>
                    <input type="text" id="field_key_path" value="${configData.key_path || ''}" placeholder="SSH私钥文件路径（与密码二选一）">
                </div>
                <div class="form-group">
                    <label>远程路径 *</label>
                    <input type="text" id="field_remote_path" value="${configData.remote_path || '/uploads'}" required placeholder="例如：/uploads">
                </div>
                <div class="form-group">
                    <label>超时时间（秒）</label>
                    <input type="number" id="field_timeout" value="${configData.timeout || 30}" min="5" max="300">
                </div>
            `;

        case 'webdav':
            return `
                <div class="form-group">
                    <label>WebDAV服务器地址 *</label>
                    <input type="text" id="field_endpoint" value="${configData.endpoint || ''}" required placeholder="例如：https://dav.jianguoyun.com/dav/">
                </div>
                <div class="form-group">
                    <label>用户名 *</label>
                    <input type="text" id="field_username" value="${configData.username || ''}" required>
                </div>
                <div class="form-group">
                    <label>密码 *</label>
                    <input type="password" id="field_password" value="${configData.password || ''}" required>
                </div>
                <div class="form-group">
                    <label>远程路径 *</label>
                    <input type="text" id="field_remote_path" value="${configData.remote_path || '/uploads'}" required>
                </div>
                <div class="form-group">
                    <label>
                        <input type="checkbox" id="field_use_https" ${configData.use_https ? 'checked' : ''}>
                        使用HTTPS
                    </label>
                </div>
                <div class="form-group">
                    <label>超时时间（秒）</label>
                    <input type="number" id="field_timeout" value="${configData.timeout || 30}" min="5" max="300">
                </div>
            `;

        case 'aliyun_oss':
            return `
                <div class="form-group">
                    <label>Endpoint *</label>
                    <input type="text" id="field_endpoint" value="${configData.endpoint || ''}" required placeholder="例如：oss-cn-hangzhou.aliyuncs.com">
                </div>
                <div class="form-group">
                    <label>AccessKey ID *</label>
                    <input type="text" id="field_access_key_id" value="${configData.access_key_id || ''}" required>
                </div>
                <div class="form-group">
                    <label>AccessKey Secret *</label>
                    <input type="password" id="field_access_key_secret" value="${configData.access_key_secret || ''}" required>
                </div>
                <div class="form-group">
                    <label>Bucket名称 *</label>
                    <input type="text" id="field_bucket_name" value="${configData.bucket_name || ''}" required>
                </div>
                <div class="form-group">
                    <label>基础路径</label>
                    <input type="text" id="field_base_path" value="${configData.base_path || ''}" placeholder="例如：uploads/">
                </div>
                <div class="form-group">
                    <label>
                        <input type="checkbox" id="field_use_https" ${configData.use_https !== false ? 'checked' : ''}>
                        使用HTTPS
                    </label>
                </div>
            `;

        case 'tencent_cos':
            return `
                <div class="form-group">
                    <label>地域 *</label>
                    <input type="text" id="field_region" value="${configData.region || ''}" required placeholder="例如：ap-guangzhou">
                </div>
                <div class="form-group">
                    <label>SecretId *</label>
                    <input type="text" id="field_secret_id" value="${configData.secret_id || ''}" required>
                </div>
                <div class="form-group">
                    <label>SecretKey *</label>
                    <input type="password" id="field_secret_key" value="${configData.secret_key || ''}" required>
                </div>
                <div class="form-group">
                    <label>Bucket名称 *</label>
                    <input type="text" id="field_bucket_name" value="${configData.bucket_name || ''}" required placeholder="格式：BucketName-APPID">
                </div>
                <div class="form-group">
                    <label>基础路径</label>
                    <input type="text" id="field_base_path" value="${configData.base_path || ''}" placeholder="例如：uploads/">
                </div>
            `;

        case 'aws_s3':
            return `
                <div class="form-group">
                    <label>地域 *</label>
                    <input type="text" id="field_region" value="${configData.region || ''}" required placeholder="例如：us-east-1">
                </div>
                <div class="form-group">
                    <label>Access Key ID *</label>
                    <input type="text" id="field_access_key_id" value="${configData.access_key_id || ''}" required>
                </div>
                <div class="form-group">
                    <label>Secret Access Key *</label>
                    <input type="password" id="field_secret_access_key" value="${configData.secret_access_key || ''}" required>
                </div>
                <div class="form-group">
                    <label>Bucket名称 *</label>
                    <input type="text" id="field_bucket_name" value="${configData.bucket_name || ''}" required>
                </div>
                <div class="form-group">
                    <label>基础路径</label>
                    <input type="text" id="field_base_path" value="${configData.base_path || ''}" placeholder="例如：uploads/">
                </div>
                <div class="form-group">
                    <label>自定义Endpoint</label>
                    <input type="text" id="field_endpoint" value="${configData.endpoint || ''}" placeholder="兼容S3的其他服务地址">
                </div>
            `;

        case 'smb':
            return `
                <div class="form-group">
                    <label>服务器地址 *</label>
                    <input type="text" id="field_host" value="${configData.host || ''}" required placeholder="例如：192.168.1.100">
                </div>
                <div class="form-group">
                    <label>端口</label>
                    <input type="number" id="field_port" value="${configData.port || 445}" min="1" max="65535">
                </div>
                <div class="form-group">
                    <label>共享名称 *</label>
                    <input type="text" id="field_share_name" value="${configData.share_name || ''}" required placeholder="例如：Public">
                </div>
                <div class="form-group">
                    <label>用户名</label>
                    <input type="text" id="field_username" value="${configData.username || ''}" placeholder="匿名访问可留空">
                </div>
                <div class="form-group">
                    <label>密码</label>
                    <input type="password" id="field_password" value="${configData.password || ''}">
                </div>
                <div class="form-group">
                    <label>域</label>
                    <input type="text" id="field_domain" value="${configData.domain || ''}" placeholder="可选">
                </div>
                <div class="form-group">
                    <label>远程路径 *</label>
                    <input type="text" id="field_remote_path" value="${configData.remote_path || '/uploads'}" required>
                </div>
            `;

        case 'local':
            return `
                <div class="form-group">
                    <label>本地存储路径 *</label>
                    <input type="text" id="field_base_path" value="${configData.base_path || ''}" required placeholder="例如：C:/Uploads 或 /var/uploads">
                </div>
                <div class="form-group">
                    <label>
                        <input type="checkbox" id="field_create_dir" ${configData.create_dir ? 'checked' : ''}>
                        路径不存在时自动创建
                    </label>
                </div>
            `;

        default:
            return '<p style="color: #999;">未知的存储类型</p>';
    }
}

// 收集表单中的配置数据
function collectStorageConfigData() {
    const storageType = document.getElementById('storageType').value;
    const configData = {};

    // 根据不同存储类型收集字段
    const fields = document.querySelectorAll('#storageConfigFields input, #storageConfigFields select');
    fields.forEach(field => {
        const fieldName = field.id.replace('field_', '');
        if (field.type === 'checkbox') {
            configData[fieldName] = field.checked;
        } else if (field.type === 'number') {
            configData[fieldName] = parseInt(field.value) || 0;
        } else {
            configData[fieldName] = field.value;
        }
    });

    return JSON.stringify(configData);
}

// 保存存储配置
function saveStorageConfig(event) {
    event.preventDefault();

    const configId = document.getElementById('configId').value;
    const name = document.getElementById('configName').value.trim();
    const storageType = document.getElementById('storageType').value;
    const description = document.getElementById('configDescription').value.trim();
    const targetAgent = document.getElementById('targetAgent').value.trim();
    const priority = parseInt(document.getElementById('configPriority').value) || 0;
    const isActive = document.getElementById('configActive').checked;

    if (!name || !storageType) {
        showStorageConfigError('请填写配置名称和存储类型');
        return;
    }

    const configData = collectStorageConfigData();

    const payload = {
        name: name,
        storage_type: storageType,
        description: description,
        target_agent_id: targetAgent || null,
        priority: priority,
        is_active: isActive,
        config_data: configData
    };

    const url = configId ? `${STORAGE_API_BASE}/${configId}` : STORAGE_API_BASE;
    const method = configId ? 'PUT' : 'POST';

    fetch(url, {
        method: method,
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify(payload)
    })
    .then(r => r.json())
    .then(data => {
        if (data.code === 200) {
            alert(configId ? '更新成功' : '创建成功');
            closeStorageConfigModal();
            loadStorageConfigs();
        } else {
            showStorageConfigError(data.message || '保存失败');
        }
    })
    .catch(err => {
        console.error('Error saving config:', err);
        showStorageConfigError('保存失败，请稍后重试');
    });
}

// 显示存储配置错误
function showStorageConfigError(message) {
    const errorDiv = document.getElementById('storageConfigError');
    errorDiv.textContent = message;
    errorDiv.style.display = 'block';
}

// HTML转义
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// 初始化存储配置管理
document.addEventListener('DOMContentLoaded', function() {
    // 加载存储类型
    loadStorageTypes();

    // 绑定存储配置表单提交
    const storageConfigForm = document.getElementById('storageConfigForm');
    if (storageConfigForm) {
        storageConfigForm.addEventListener('submit', saveStorageConfig);
    }

    // 点击模态框外部关闭
    window.addEventListener('click', function(e) {
        const modal = document.getElementById('storageConfigModal');
        if (e.target === modal) {
            closeStorageConfigModal();
        }
    });
});
