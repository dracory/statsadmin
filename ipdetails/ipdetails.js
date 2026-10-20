const { createApp, ref, computed, onMounted } = Vue;

// esc escapes a string for safe insertion into HTML innerHTML.
// Prevents XSS when displaying user-controlled data (paths, patterns) in
// Notiflix.Report dialogs.
const esc = (s) => String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');

createApp({
    setup() {
        const conditions = ref([]);

        if (localStorage.getItem('stats_include_bots') === 'false') {
            if (!conditions.value.some(c => c.field === 'is_bot' && c.value === 'no')) {
                conditions.value.push({ field: 'is_bot', operator: 'equals', value: 'no' });
            }
        }

        const includeBots = computed(() => {
            return !conditions.value.some(c => c.field === 'is_bot' && c.value === 'no');
        });

        const details = ref({});
        const paths = ref([]);
        const visitCount = ref(0);
        const isBot = ref(false);
        const isThreat = ref(false);
        const botReasons = ref([]);
        const loading = ref(true);
        const flagging = ref(false);
        const flaggingThreat = ref(false);
        const removing = ref(false);

        const toggleBots = () => {
            if (includeBots.value) {
                if (!conditions.value.some(c => c.field === 'is_bot' && c.value === 'no')) {
                    conditions.value.push({ field: 'is_bot', operator: 'equals', value: 'no' });
                }
                localStorage.setItem('stats_include_bots', 'false');
            } else {
                conditions.value = conditions.value.filter(c => c.field !== 'is_bot');
                localStorage.setItem('stats_include_bots', 'true');
            }
            loadDetails();
        };

        const loadDetails = async () => {
            try {
                const response = await fetch(urlLoadPaths, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ ip: '__IP__', conditions: conditions.value }),
                });
                const data = await response.json();
                if (data.status === 'success') {
                    details.value = data.data.details || {};
                    paths.value = data.data.paths || [];
                    visitCount.value = data.data.visit_count || 0;
                    isBot.value = data.data.is_bot || false;
                    isThreat.value = data.data.is_threat || false;
                    botReasons.value = data.data.bot_reasons || [];
                } else {
                    Notiflix.Notify.failure(data.message || 'Failed to load IP details');
                }
            } catch (e) {
                Notiflix.Notify.failure('Failed to load IP details');
            } finally {
                loading.value = false;
            }
        };

        const confirmFlagBot = () => {
            Notiflix.Confirm.show(
                'Flag as Bot',
                'Are you sure you want to flag ' + (details.value.ip || '__IP__') + ' as a bot? This IP will be excluded from visitor statistics.',
                'Yes, flag it',
                'Cancel',
                () => flagBot(),
                () => {}
            );
        };

        // showBotReasons displays the bot reasons in a Notiflix.Report dialog.
        // The reasons are pre-loaded with the IP details (no AJAX needed).
        const showBotReasons = () => {
            const ip = details.value.ip || '__IP__';
            const reasons = botReasons.value || [];
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
        };

        const flagBot = async () => {
            flagging.value = true;
            try {
                const response = await fetch(urlFlagBot, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ ip: '__IP__' }),
                });
                const data = await response.json();
                if (data.status === 'success') {
                    isBot.value = true;
                    Notiflix.Notify.success('IP flagged as bot');
                } else {
                    Notiflix.Notify.failure(data.message || 'Failed to flag IP');
                }
            } catch (e) {
                Notiflix.Notify.failure('Failed to flag IP');
            } finally {
                flagging.value = false;
            }
        };

        const confirmFlagThreat = () => {
            Notiflix.Confirm.show(
                'Flag as Threat',
                'Are you sure you want to flag ' + (details.value.ip || '__IP__') + ' as a threat? All visitor records from this IP will be marked as threat=yes.',
                'Yes, flag it',
                'Cancel',
                () => flagThreat(),
                () => {}
            );
        };

        const flagThreat = async () => {
            flaggingThreat.value = true;
            try {
                const response = await fetch(urlFlagThreat, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ ip: '__IP__' }),
                });
                const data = await response.json();
                if (data.status === 'success') {
                    isThreat.value = true;
                    Notiflix.Notify.success('IP flagged as threat');
                } else {
                    Notiflix.Notify.failure(data.message || 'Failed to flag IP');
                }
            } catch (e) {
                Notiflix.Notify.failure('Failed to flag IP');
            } finally {
                flaggingThreat.value = false;
            }
        };

        onMounted(() => {
            loadDetails();
        });

        const confirmRemoveEntries = () => {
            Notiflix.Confirm.show(
                'Remove Entries',
                'Are you sure you want to permanently delete ALL visitor records and path history for ' + (details.value.ip || '__IP__') + '? This action cannot be undone.',
                'Yes, delete all',
                'Cancel',
                () => removeEntries(),
                () => {}
            );
        };

        const removeEntries = async () => {
            removing.value = true;
            try {
                const response = await fetch(urlRemoveEntries, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ ip: '__IP__' }),
                });
                const data = await response.json();
                if (data.status === 'success') {
                    Notiflix.Notify.success(data.message || 'Entries removed');
                    // Clear the local view since the records no longer exist.
                    details.value = {};
                    paths.value = [];
                    visitCount.value = 0;
                    isBot.value = false;
                    isThreat.value = false;
                    botReasons.value = [];
                } else {
                    Notiflix.Notify.failure(data.message || 'Failed to remove entries');
                }
            } catch (e) {
                Notiflix.Notify.failure('Failed to remove entries');
            } finally {
                removing.value = false;
            }
        };

        // pathUrl strips the "[METHOD] " prefix from a stored path (e.g. "[GET] /foo")
        // and builds an absolute URL so the link can open in a new tab.
        const pathUrl = (path) => {
            if (!path) return '#';
            const stripped = String(path).replace(/^\[[^\]]+\]\s*/, '');
            if (!stripped.startsWith('/')) return stripped;
            return window.location.origin + stripped;
        };

        return {
            includeBots,
            toggleBots,
            details,
            paths,
            visitCount,
            isBot,
            isThreat,
            botReasons,
            loading,
            flagging,
            flaggingThreat,
            removing,
            confirmFlagBot,
            confirmFlagThreat,
            confirmRemoveEntries,
            showBotReasons,
            pathUrl,
        };
    }
}).mount('#ip-details-app');
