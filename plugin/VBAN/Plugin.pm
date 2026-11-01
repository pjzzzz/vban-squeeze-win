# VBAN, a plugin to play VBAN streams on Lyrion Music Server (Windows port)
#
# Copyright (C) 2017 by Markus Buschhoff
# Windows port (C) 2026 by pjzzzz
# Based on WaveInput, (C) 2008 Bryan Alton and others
#
# This program is free software; you can redistribute it and/or
# modify it under the same terms as Perl itself.

package Plugins::VBAN::Plugin;

use strict;

use base qw(Slim::Plugin::OPMLBased);

use File::Basename qw(dirname);
use File::Spec::Functions qw(catdir catfile);

use Slim::Utils::Log;
use Slim::Utils::Misc;
use Slim::Utils::OSDetect;
use Slim::Utils::Prefs;
use Slim::Utils::Strings qw(string);

use Slim::Control::Request;
use Slim::Music::Info;
use Slim::Player::Client;
use Slim::Player::Playlist;

use Plugins::VBAN::Settings;
use Plugins::VBAN::VBAN;

# Create the log category before loading the protocol handler.
my $log = Slim::Utils::Log->addLogCategory({
	'category'     => 'plugin.VBAN',
	'defaultLevel' => 'INFO',
	'description'  => getDisplayName(),
});

my $prefs = preferences('plugin.VBAN');

################################
### Plugin Interface ###########
################################

sub initPlugin {
	my $class = shift;

	$prefs->init({
		host   => '',
		port   => 6980,
		stream => 'Stream1',
	});

	$class->_setupBinPath();

	Plugins::VBAN::Settings->new;

	# CLI: ["vban","title","<text>"] lets a sender (for example the MediaMonkey
	# addon) push the current track without a separate VBAN TEXT sender.
	Slim::Control::Request::addDispatch(['vban', 'title', '_'], [0, 0, 1, \&_cliSetTitle]);

	$class->SUPER::initPlugin(
		feed   => \&handleFeed,
		tag    => 'vban',
		is_app => 1,
	);

	$log->info('Initialising ' . $class->_pluginDataFor('version') . ' on ' . $^O);

	return 1;
}

sub shutdownPlugin {
	return;
}

sub getDisplayName {
	return ('PLUGIN_VBAN');
}

# CLI: ["vban","title","<text>"] - set the now-playing title for VBAN streams.
sub _cliSetTitle {
	my $request = shift;
	my $title   = $request->getParam('_');

	unless (defined $title && length $title) {
		$request->setStatusBadParams();
		return;
	}

	$title =~ s/[\x00-\x1f\x7f]//g;
	$title = substr($title, 0, 512);

	my $raw = $title;
	my ($artist, $album) = ('', '');

	if ($raw =~ /\|/) {
		($artist, $title, $album) = split /\|/, $raw, 3;
	}
	elsif ($raw =~ /^(.+?) - (.+)$/) {
		($artist, $title) = ($1, $2);
	}

	$title  = '' unless defined $title;
	$artist = '' unless defined $artist;
	$album  = '' unless defined $album;

	my $stream = $prefs->get('stream') || 'Stream1';
	(my $safe = $stream) =~ s/[^A-Za-z0-9._-]/_/g;
	$safe = 'Stream1' unless length $safe;

	_write_meta($safe, $title, $artist, $album);

	my $file = 'C:\ProgramData\Lyrion\vban-title-' . $safe . '.txt';
	my $tmp  = $file . '.tmp';
	if (open my $fh, '>:encoding(UTF-8)', $tmp) {
		print $fh $title;
		close $fh;
		rename($tmp, $file) or $log->error("vban: cannot rename $tmp to $file: $!");
	}
	else {
		$log->error("vban: cannot write title file $tmp: $!");
	}

	for my $client (Slim::Player::Client::clients()) {
		my $url = Slim::Player::Playlist::url($client) || '';
		next unless $url =~ m{^vban://[^/]+/\Q$stream\E$}i;
		Slim::Music::Info::setTitle($url, $title);
		Slim::Music::Info::setCurrentTitle($url, $title, $client);
	}

	$request->setStatusDone();
}

sub _write_meta {
	my ($safe, $title, $artist, $album) = @_;

	my $file = 'C:\ProgramData\Lyrion\vban-meta-' . $safe . '.txt';
	my $tmp  = $file . '.tmp';

	if (open my $fh, '>:encoding(UTF-8)', $tmp) {
		print $fh join("\n", $title, $artist, $album);
		close $fh;
		rename($tmp, $file) or $log->error("vban: cannot rename $tmp to $file: $!");
	}
	else {
		$log->error("vban: cannot write metadata file $tmp: $!");
	}
}

# Radio menu entry: one item for the configured VBAN stream.
sub handleFeed {	my ($client, $cb, $args, $params) = @_;

	my $host   = $prefs->get('host');
	my $port   = $prefs->get('port') || 6980;
	my $stream = $prefs->get('stream') || 'Stream1';

	my @items;

	if ($host) {
		my $url = 'vban://' . $host
			. ($port == 6980 ? '' : ":$port")
			. '/' . $stream;

		# Same whitelist as the protocol handler: never offer a URL that could
		# act as a shell metacharacter or a malformed stream name.
		if (Plugins::VBAN::VBAN->_valid_url($url)) {
			push @items, {
				name     => $stream,
				type     => 'audio',
				url      => $url,
				icon     => Plugins::VBAN::Plugin->_pluginDataFor('icon'),
				onselect => 'play',
			};
		}
	}

	unless (@items) {
		push @items, {
			name => string('PLUGIN_VBAN_CONFIGURE'),
			type => 'text',
		};
	}

	$cb->({ items => \@items });
}

# Lyrion already adds plugin/Bin/<binArch> to its search path, but the
# transcoder is started through socketwrapper.exe and resolves the command via
# PATH. Set both explicitly so vban_url_receptor.exe is always found.
sub _setupBinPath {
	my $class = shift;

	my $pluginBase = eval { $class->_pluginDataFor('basedir') };
	my $base       = catdir($pluginBase || dirname(__FILE__), 'Bin');
	unless (-d $base) {
		$log->error("VBAN Bin directory not found: $base");
		return;
	}

	my $details = eval { Slim::Utils::OSDetect::details() };
	my $arch    = ref($details) eq 'HASH' ? $details->{'binArch'} : undef;

	my @paths = grep { defined $_ && -d $_ } (
		($arch ? catdir($base, $arch) : ()),
		catdir($base, $^O),
		$base,
	);

	unless (@paths) {
		$log->error("no VBAN binary directory found below $base");
		return;
	}

	eval { Slim::Utils::Misc::addFindBinPaths(@paths) };

	$ENV{PATH} = join(';', @paths) . ';' . ($ENV{PATH} || '');

	$log->info('VBAN binary path: ' . join(';', @paths));

	my $exe = catfile($paths[0], 'vban_url_receptor.exe');
	$log->error("vban_url_receptor.exe not found in $paths[0]") unless -f $exe;
}

1;
