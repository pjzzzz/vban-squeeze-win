(() => {
    'use strict';

    const SETTINGS_KEY = 'mmSyncLyrionBridge_settings';

    // No server address, player MAC or stream URL is shipped as a default:
    // fill these in on this page before enabling the bridge.
    const DEFAULT_SETTINGS = {
        enabled: true,
        lyrionHost: '',
        lyrionPort: '9002',
        playerMac: '',
        streamUrl: '',
        powerOffOnStop: false,
        vbanTitleEnabled: true,
        vbanStream: 'Stream1'
    };

    function isValidStreamUrl(url) {
        return /^https?:\/\/([^:@/\s]+)(:\d+)?\/stream\/swyh\.(rf64|flac)(\?.*)?$/i.test(url);
    }

    function isValidMac(mac) {
        return /^([0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}$/.test(mac);
    }

    // Host name or IPv4 address; no scheme, port, path or whitespace.
    function isValidHost(host) {
        return /^[A-Za-z0-9]([A-Za-z0-9.-]*[A-Za-z0-9])?$/.test(host);
    }

    function readSettings(UI) {
        return {
            enabled: UI.chbEnabled.controlClass.checked,
            lyrionHost: UI.editLyrionHost.controlClass.value.trim(),
            lyrionPort: UI.editLyrionPort.controlClass.value.trim(),
            playerMac: UI.editPlayerMac.controlClass.value.trim(),
            streamUrl: UI.editStreamUrl.controlClass.value.trim(),
            powerOffOnStop: UI.chbPowerOffOnStop.controlClass.checked,
            vbanTitleEnabled: UI.chbVbanTitle.controlClass.checked,
            vbanStream: UI.editVbanStream.controlClass.value.trim()
        };
    }

    function validateSettings(settings) {
        if (!settings.lyrionHost) {
            return 'Server address must not be empty';
        }
        if (!isValidHost(settings.lyrionHost)) {
            return 'Server address must be a host name or IPv4 address (letters, digits, dots, hyphens)';
        }
        const port = parseInt(settings.lyrionPort, 10);
        if (!settings.lyrionPort || isNaN(port) || port < 1 || port > 65535) {
            return 'Server port must be a number between 1 and 65535';
        }
        if (!isValidMac(settings.playerMac)) {
            return 'Invalid player MAC (expected AA:BB:CC:DD:EE:FF)';
        }
        if (!isValidStreamUrl(settings.streamUrl)) {
            return 'Invalid stream URL (expected http(s)://host/stream/swyh.rf64 or .flac)';
        }
        if (settings.vbanTitleEnabled && (!settings.vbanStream || settings.vbanStream.length > 16)) {
            return 'VBAN stream name must be 1-16 characters (for example Stream1)';
        }
        return null;
    }

    function setStatus(panel, message, isError) {
        const el = panel.querySelector('#lblStatus');
        if (el) {
            el.textContent = message;
            el.style.color = isError ? '#c0392b' : '#27ae60';
        }
    }

    function testConnection(panel, UI) {
        const settings = readSettings(UI);
        const error = validateSettings(settings);
        if (error) {
            setStatus(panel, error, true);
            return;
        }
        setStatus(panel, 'Connecting...', false);
        const payload = {
            id: 1,
            method: 'slim.request',
            params: [settings.playerMac, ['status', '0', '1']]
        };
        app.utils.web.getURLContentAsync('http://' + settings.lyrionHost + ':' + settings.lyrionPort + '/jsonrpc.js', {
            headers: ['Content-Type: application/json'],
            requestBody: JSON.stringify(payload)
        }).then(function (responseText) {
            try {
                const resp = JSON.parse(responseText);
                if (resp && resp.result && resp.result.mode) {
                    setStatus(panel, 'Connection OK (player: ' + resp.result.mode + ')', false);
                } else {
                    setStatus(panel, 'Connection OK, but no player status received', true);
                }
            } catch (e) {
                setStatus(panel, 'Connection OK, but the response was not valid JSON', true);
            }
        }).catch(function (err) {
            setStatus(panel, 'Connection failed: ' + err, true);
        });
    }

    function testStream(panel, UI) {
        const settings = readSettings(UI);
        const error = validateSettings(settings);
        if (error) {
            setStatus(panel, error, true);
            return;
        }
        setStatus(panel, 'Testing stream...', false);
        app.utils.web.requestAsync({
            method: 'GET',
            uri: settings.streamUrl,
            headers: [],
            doneCallback: function (status) {
                if (status >= 200 && status < 300) {
                    setStatus(panel, 'Stream reachable (status ' + status + ')', false);
                } else {
                    setStatus(panel, 'Stream not reachable (status ' + status + ')', true);
                }
            }
        });
    }

    window.configInfo = {
        load: function (panel, addon) {
            const settings = app.getValue(SETTINGS_KEY, DEFAULT_SETTINGS);
            const UI = getAllUIElements(panel);

            UI.chbEnabled.controlClass.checked = settings.enabled;
            UI.editLyrionHost.controlClass.value = settings.lyrionHost;
            UI.editLyrionPort.controlClass.value = settings.lyrionPort;
            UI.editPlayerMac.controlClass.value = settings.playerMac;
            UI.editStreamUrl.controlClass.value = settings.streamUrl;
            UI.chbPowerOffOnStop.controlClass.checked = settings.powerOffOnStop;
            UI.chbVbanTitle.controlClass.checked = settings.vbanTitleEnabled;
            UI.editVbanStream.controlClass.value = settings.vbanStream;

            app.listen(UI.btnTestConnection, 'click', () => testConnection(panel, UI));
            app.listen(UI.btnTestStream, 'click', () => testStream(panel, UI));
        },

        save: function (panel, addon) {
            const UI = getAllUIElements(panel);
            const settings = readSettings(UI);

            const error = validateSettings(settings);
            if (error) {
                setStatus(panel, error, true);
                return false;
            }

            app.setValue(SETTINGS_KEY, settings);
            setStatus(panel, 'Settings saved', false);
            return true;
        }
    };
})();
