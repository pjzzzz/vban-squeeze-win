requirejs('actions');

window.whenReady(() => {
    'use strict';

    const SETTINGS_KEY = 'mmSyncLyrionBridge_settings';

    // No server address, player MAC or stream URL is shipped as a default:
    // fill these in under Tools > Options > Add-ons > MMSync > Configure.
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

    let inFlight = false;
    let pendingCommand = null;
    let currentTimeout = null;
    let titleTimeout = null;

    function getSettings() {
        return app.getValue(SETTINGS_KEY, DEFAULT_SETTINGS);
    }

    function lyrionUrl(settings) {
        return 'http://' + settings.lyrionHost + ':' + settings.lyrionPort + '/jsonrpc.js';
    }

    function isValidStreamUrl(url) {
        return /^https?:\/\/([^:@/\s]+)(:\d+)?\/stream\/swyh\.(rf64|flac)(\?.*)?$/i.test(url);
    }

    function isValidMac(mac) {
        return /^([0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}$/.test(mac);
    }

    function sendCommand(settings, command) {
        const payload = {
            id: 1,
            method: 'slim.request',
            params: [settings.playerMac, command]
        };
        if (inFlight) {
            pendingCommand = { settings: settings, command: command };
            return;
        }
        inFlight = true;
        app.utils.web.getURLContentAsync(lyrionUrl(settings), {
            headers: ['Content-Type: application/json'],
            requestBody: JSON.stringify(payload)
        }).then(function () {
            inFlight = false;
            if (pendingCommand) {
                const pc = pendingCommand;
                pendingCommand = null;
                sendCommand(pc.settings, pc.command);
            }
        }).catch(function (err) {
            console.error('MMSync: JSON-RPC request failed', err);
            inFlight = false;
            if (pendingCommand) {
                const pc = pendingCommand;
                pendingCommand = null;
                sendCommand(pc.settings, pc.command);
            }
        });
    }

    function onPlay(settings) {
        if (!isValidStreamUrl(settings.streamUrl)) {
            console.warn('MMSync: invalid streamUrl, not starting playback');
            return;
        }
        sendCommand(settings, ['playlist', 'play', settings.streamUrl]);
    }

    function onPause(settings) {
        sendCommand(settings, ['pause']);
    }

    function onUnpause(settings) {
        sendCommand(settings, ['play']);
    }

    function onStop(settings) {
        if (settings.powerOffOnStop) {
            sendCommand(settings, ['power', 0]);
        } else {
            sendCommand(settings, ['stop']);
        }
    }

    function sendTitle(settings) {
        if (!settings.vbanTitleEnabled) {
            return;
        }
        const track = app.player.getCurrentTrack();
        if (!track || !track.title) {
            return;
        }
        if (!settings.lyrionHost || !settings.lyrionPort) {
            console.warn('MMSync: no Lyrion server configured, title not sent');
            return;
        }
        // The Lyrion plugin splits the text on '|', so a pipe inside a field
        // would shift artist/title/album. Replace it before joining.
        const artist = (track.artist || '').replace(/\|/g, '/').replace(/[\r\n]+/g, ' ').trim();
        const album = (track.album || '').replace(/\|/g, '/').replace(/[\r\n]+/g, ' ').trim();
        const title = String(track.title).replace(/\|/g, '/').replace(/[\r\n]+/g, ' ').trim();
        const text = artist + '|' + title + '|' + album;
        sendCommand(settings, ['vban', 'title', text]);
    }

    function onPlaybackStateChange(newState) {
        const settings = getSettings();

        if ((newState === 'play' || newState === 'trackChanged') && settings.vbanTitleEnabled) {
            clearTimeout(titleTimeout);
            titleTimeout = requestTimeout(() => sendTitle(settings), 50);
        }

        if (!settings.enabled) {
            return;
        }
        // Nothing to talk to yet: do not build http://:9002/jsonrpc.js.
        if (!settings.lyrionHost || !settings.lyrionPort) {
            console.warn('MMSync: no Lyrion server configured, not starting');
            return;
        }
        if (!isValidMac(settings.playerMac)) {
            console.warn('MMSync: invalid playerMac, not starting');
            return;
        }
        clearTimeout(currentTimeout);
        switch (newState) {
            case 'play':
                currentTimeout = requestTimeout(() => onPlay(settings), 50);
                break;
            case 'pause':
                currentTimeout = requestTimeout(() => onPause(settings), 50);
                break;
            case 'unpause':
                currentTimeout = requestTimeout(() => onUnpause(settings), 50);
                break;
            case 'stop':
                currentTimeout = requestTimeout(() => onStop(settings), 50);
                break;
        }
    }

    app.listen(app.player, 'playbackState', onPlaybackStateChange);
});