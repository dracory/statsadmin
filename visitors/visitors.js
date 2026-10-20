const { createApp, ref, computed, onMounted } = Vue;

// esc escapes a string for safe insertion into HTML innerHTML.
// Prevents XSS when displaying user-controlled data (paths, patterns) in
// Notiflix.Report dialogs.
const esc = (s) => String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');

const conditionOptions = [
    { value: 'ip',          label: 'IP Address',        operators: ['equals'],    inputType: 'text',   placeholder: 'e.g. 192.168.1.1' },
    { value: 'country',     label: 'Country',           operators: ['equals'],    inputType: 'text',   placeholder: 'ISO2 code (e.g. GB)' },
    { value: 'device_type', label: 'Device Type',       operators: ['equals'],    inputType: 'select', placeholder: '' },
    { value: 'path_contains', label: 'Path contains',   operators: ['contains'],  inputType: 'text',   placeholder: 'e.g. /admin' },
    { value: 'path_exact',  label: 'Path equals',       operators: ['equals'],    inputType: 'text',   placeholder: 'e.g. [GET] /' },
    { value: 'browser',     label: 'Browser',           operators: ['equals'],    inputType: 'text',   placeholder: 'e.g. Chrome' },
    { value: 'os',          label: 'Operating System',  operators: ['equals'],    inputType: 'text',   placeholder: 'e.g. Windows' },
    { value: 'date_from',   label: 'Created after',     operators: ['equals'],    inputType: 'date',   placeholder: '' },
    { value: 'date_to',     label: 'Created before',    operators: ['equals'],    inputType: 'date',   placeholder: '' },
    { value: 'is_bot',      label: 'Is Bot',            operators: ['equals'],    inputType: 'select', placeholder: '' },
];

const deviceTypes = ['desktop', 'mobile', 'tablet', 'bot'];

const opLabels = { equals: '=', contains: 'contains' };

function getFieldOpt(value) {
    return conditionOptions.find(o => o.value === value);
}

createApp({
    setup() {
        const visitors = ref([]);
        const total = ref(0);
        const page = ref(1);
        const perPage = ref(25);
        const totalPages = ref(0);

        // Applied conditions (sent to the server)
        const conditions = ref([]);

        const includeBots = computed(() => {
            return !conditions.value.some(c => c.field === 'is_bot' && c.value === 'no');
        });

        const toggleBots = () => {
            if (includeBots.value) {
                // Currently ON -> turn OFF (add is_bot=no)
                if (!conditions.value.some(c => c.field === 'is_bot' && c.value === 'no')) {
                    conditions.value.push({ field: 'is_bot', operator: 'equals', value: 'no' });
                }
                localStorage.setItem('stats_include_bots', 'false');
            } else {
                // Currently OFF -> turn ON (remove is_bot conditions)
                conditions.value = conditions.value.filter(c => c.field !== 'is_bot');
                localStorage.setItem('stats_include_bots', 'true');
            }
            page.value = 1;
            loadVisitors();
            updateURL();
        };

        // Modal state
        const showFilterModal = ref(false);
        const modalConditions = ref([]);

        const openFilterModal = () => {
            // Clone current conditions into the modal editor
            modalConditions.value = conditions.value.map(c => ({ ...c }));
            if (modalConditions.value.length === 0) {
                modalConditions.value.push({ field: '', operator: 'equals', value: '' });
            }
            showFilterModal.value = true;
        };

        const addModalCondition = () => {
            modalConditions.value.push({ field: '', operator: 'equals', value: '' });
        };

        const removeModalCondition = (i) => {
            modalConditions.value.splice(i, 1);
        };

        const clearModalConditions = () => {
            modalConditions.value = [];
        };

        const onFieldChange = (c) => {
            const opt = getFieldOpt(c.field);
            if (opt) {
                c.operator = opt.operators[0];
            }
            c.value = '';
        };

        const applyConditions = () => {
            // Filter out empty conditions
            conditions.value = modalConditions.value.filter(c => c.field && c.value.trim());
            showFilterModal.value = false;
            page.value = 1;
            loadVisitors();
            updateURL();
        };

        const clearConditions = () => {
            conditions.value = [];
            page.value = 1;
            loadVisitors();
            updateURL();
        };

        const removeCondition = (i) => {
            conditions.value.splice(i, 1);
            page.value = 1;
            loadVisitors();
            updateURL();
        };

        const exportCSV = () => {
            const form = document.createElement('form');
            form.method = 'POST';
            form.action = urlVisitorsBase + '&action=export-csv';
            
            const input = document.createElement('input');
            input.type = 'hidden';
            input.name = 'conditions'; // The backend expect conditions in JSON body, but standard form POST is different.
            // Actually, my backend handleExportCSV tries JSON decoder first.
            // Standard form POST doesn't send JSON body easily.
            // I'll change handleExportCSV to support query params or standard form.
            
            // Re-evaluating: easiest is to use fetch and create a blob.
            doExport();
        };

        const doExport = async () => {
            try {
                const response = await fetch(urlVisitorsBase + '&action=export-csv', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ conditions: conditions.value }),
                });
                const blob = await response.blob();
                const url = window.URL.createObjectURL(blob);
                const a = document.createElement('a');
                a.href = url;
                a.download = 'visitors.csv';
                document.body.appendChild(a);
                a.click();
                window.URL.revokeObjectURL(url);
            } catch (error) {
                Notiflix.Notify.failure('Failed to export CSV');
            }
        };

        const loadVisitors = async () => {
            try {
                const response = await fetch(urlLoadVisitors, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        page: page.value,
                        per_page: perPage.value,
                        conditions: conditions.value,
                    }),
                });
                const data = await response.json();
                if (data.status === 'success') {
                    visitors.value = data.data.visitors || [];
                    total.value = data.data.total || 0;
                    totalPages.value = data.data.total_pages || 0;
                } else {
                    Notiflix.Notify.failure(data.message || 'Failed to load visitors');
                }
            } catch (error) {
                Notiflix.Notify.failure('Failed to load visitors');
            }
        };

        const prevPage = () => {
            if (page.value > 1) {
                page.value--;
                loadVisitors();
                updateURL();
            }
        };

        // showBotReasons fetches the bot reasons for the given IP via AJAX
        // from the ipdetails controller's load-bot-reasons action and
        // displays them in a Notiflix.Report dialog.
        const showBotReasons = async (ip) => {
            try {
                const url = urlIPDetailsBase + '&action=load-bot-reasons&ip=' + encodeURIComponent(ip);
                const response = await fetch(url, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ ip: ip }),
                });
                const data = await response.json();
                if (data.status !== 'success') {
                    Notiflix.Notify.failure(data.message || 'Failed to load bot reasons');
                    return;
                }
                const reasons = data.data.bot_reasons || [];
                let html;
                if (reasons.length === 0) {
                    html = '<p style="margin:0;color:#6c757d;">No specific reasons recorded. This IP may have been flagged manually.</p>';
                } else {
                    html = '<div style="text-align:left;">';
                    for (const r of reasons) {
                        html += '<div style="margin-bottom:8px;">';
                        html += '<span style="background:#f8f9fa;border:1px solid #dee2e6;padding:2px 8px;border-radius:4px;font-size:12px;">' + esc(r.pattern) + '</span>';
                        html += '<span style="color:#6c757d;margin-left:6px;font-size:12px;">' + r.hits + 'x</span>';
                        if (r.paths && r.paths.length) {
                            html += '<div style="color:#6c757d;font-size:11px;margin-left:16px;margin-top:4px;font-family:monospace;">';
                            for (const p of r.paths) {
                                html += '<div>' + esc(p) + '</div>';
                            }
                            html += '</div>';
                        }
                        html += '</div>';
                    }
                    html += '</div>';
                }
                Notiflix.Report.info(
                    'Bot Reasons — ' + esc(ip),
                    html,
                    'Close'
                );
            } catch (e) {
                Notiflix.Notify.failure('Failed to load bot reasons');
            }
        };

        const nextPage = () => {
            if (page.value < totalPages.value) {
                page.value++;
                loadVisitors();
                updateURL();
            }
        };

        // Encode/decode filter conditions to/from URL query string so that
        // saved links restore the active filters.
        const updateURL = () => {
            const params = new URLSearchParams();
            if (conditions.value.length > 0) {
                params.set('filters', JSON.stringify(conditions.value));
            }
            if (page.value > 1) {
                params.set('page', String(page.value));
            }
            const qs = params.toString();
            const newURL = qs ? window.location.pathname + '?' + qs : window.location.pathname;
            history.replaceState(null, '', newURL);
        };

        const loadFromURL = () => {
            const params = new URLSearchParams(window.location.search);
            const filtersRaw = params.get('filters');
            let hasFiltersInURL = false;
            if (filtersRaw) {
                try {
                    const parsed = JSON.parse(filtersRaw);
                    if (Array.isArray(parsed)) {
                        conditions.value = parsed.filter(c => c && c.field && c.value);
                        hasFiltersInURL = true;
                    }
                } catch (e) { /* ignore malformed */ }
            }
            if (!hasFiltersInURL) {
                if (localStorage.getItem('stats_include_bots') === 'false') {
                    if (!conditions.value.some(c => c.field === 'is_bot')) {
                        conditions.value.push({ field: 'is_bot', operator: 'equals', value: 'no' });
                    }
                }
            }
            const pageRaw = params.get('page');
            if (pageRaw) {
                const p = parseInt(pageRaw, 10);
                if (p > 0) page.value = p;
            }
        };

        // Helper functions for the template
        const fieldLabel = (v) => { const o = getFieldOpt(v); return o ? o.label : v; };
        const opLabel = (op) => opLabels[op] || op;
        const getFieldOperators = (v) => { const o = getFieldOpt(v); return o ? o.operators : ['equals']; };
        const getFieldInputType = (v) => { const o = getFieldOpt(v); return o ? o.inputType : 'text'; };
        const getFieldPlaceholder = (v) => { const o = getFieldOpt(v); return o ? o.placeholder : ''; };
        const ipDetailsUrl = (ip) => urlIPDetailsBase + '&ip=' + encodeURIComponent(ip);
        // pathUrl strips the "[METHOD] " prefix from a stored path (e.g. "[GET] /foo")
        // and builds an absolute URL so the link can open in a new tab.
        const pathUrl = (path) => {
            if (!path) return '#';
            const stripped = String(path).replace(/^\[[^\]]+\]\s*/, '');
            if (!stripped.startsWith('/')) return stripped;
            return window.location.origin + stripped;
        };

        onMounted(() => {
            loadFromURL();
            loadVisitors();
        });

        return {
            includeBots, toggleBots,
            visitors, total, page, perPage, totalPages,
            conditions, showFilterModal, modalConditions,
            conditionOptions, deviceTypes,
            openFilterModal, addModalCondition, removeModalCondition,
            clearModalConditions, onFieldChange, applyConditions,
            clearConditions, removeCondition,
            loadVisitors, prevPage, nextPage,
            exportCSV,
            fieldLabel, opLabel, getFieldOperators, getFieldInputType, getFieldPlaceholder,
            ipDetailsUrl,
            pathUrl,
            showBotReasons,
        };
    }
}).mount('#stats-visitors-app');
