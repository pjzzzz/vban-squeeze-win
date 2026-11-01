# VBAN, a plugin to play VBAN streams on Lyrion Music Server (Windows port)
#
# Copyright (C) 2017 by Markus Buschhoff
# Windows port (C) 2026 by pjzzzz
# Based on WaveInput, (C) 2008 Bryan Alton and others
#
# This program is free software; you can redistribute it and/or
# modify it under the same terms as Perl itself.

package Plugins::VBAN::VBAN;

use strict;

use base qw(Slim::Player::Pipeline);

use Slim::Music::Info;
use Slim::Player::ProtocolHandlers;
use Slim::Player::Song;
use Slim::Player::TranscodingHelper;
use Slim::Utils::Log;
use Slim::Utils::Misc;
use Slim::Utils::Prefs;
use Slim::Utils::Strings qw(string);
use Slim::Utils::Timers;

use File::Spec;
use Time::HiRes qw(time);

Slim::Player::ProtocolHandlers->registerHandler('vban', __PACKAGE__);

my $log   = logger('plugin.VBAN');
my $prefs = preferences('server');

sub isRemote { 1 }

# Live stream: keep the pipeline buffer small so playback starts quickly.
sub bufferThreshold { 150 }

sub _valid_url {
	my ($class, $url) = @_;

	return $url =~ m{\Avban://[A-Za-z0-9.\-]{1,253}(?::\d{1,5})?(?:/[A-Za-z0-9._~%\-]{0,16})?\z}i;
}

sub new {
	my $class = shift;
	my $args  = shift;

	my $transcoder = $args->{'transcoder'};
	my $url        = $args->{'url'};
	my $client     = $args->{'client'};

	my $restoredurl = $url;
	$restoredurl =~ s|^vban:||;

	# Defence in depth: this URL ends up in a transcoder command line that
	# Lyrion hands to socketwrapper/cmd.exe. Refuse anything that could act as
	# a shell metacharacter instead of relying on quoting.
	unless ($class->_valid_url($url)) {
		$log->error("refusing invalid VBAN URL: $url");
		return;
	}

	Slim::Music::Info::setContentType($url, 'vban');

	my $quality = $prefs->client($client)->get('lameQuality');
	my $command = Slim::Player::TranscodingHelper::tokenizeConvertCommand2(
		$transcoder, $restoredurl, $url, 1, $quality
	);

	$log->debug("VBAN command = '$command'");

	unless ($command) {
		$log->error("could not build a transcoder command for $url "
			. "(is vban_url_receptor.exe present in the plugin Bin folder?)");
		return;
	}

	# Third argument = "local": skip socketwrapper's -w stream check. A VBAN
	# source can be silent for a while (VoiceMeeter stream switched off); the
	# receiver must stay alive so playback resumes when the source returns.
	my $self = $class->SUPER::new(undef, $command, 1);

	${*$self}{'contentType'} = $transcoder->{'streamformat'};

	$self->_start_title_timer($client, $url);

	return $self;
}

# VBAN TEXT packets are turned into a title file by the receiver. Poll it and
# publish the title to Lyrion so the player shows the current track.
sub _start_title_timer {
	my ($self, $client, $url) = @_;

	my $stream = $url =~ m{^vban://[^/]+/([^/?#]+)} ? $1 : 'Stream1';
	(my $safe = $stream) =~ s/[^A-Za-z0-9._-]/_/g;
	$safe = 'Stream1' unless length $safe;

	if (my $old = ${*$self}{'titleTimer'}) {
		Slim::Utils::Timers::killSpecific($old);
		delete ${*$self}{'titleTimer'};
	}

	# Per-stream file in the Lyrion data directory; the receiver uses the same
	# name, so %TEMP% differences between processes do not matter.
	my $file = 'C:\ProgramData\Lyrion\vban-title-' . $safe . '.txt';

	${*$self}{'titleFile'}  = $file;
	${*$self}{'lastTitle'}  = '';

	my $apply = sub {
		return unless -f $file;

		open my $fh, '<:encoding(UTF-8)', $file or return;
		local $/;
		my $title = <$fh>;
		close $fh;

		return unless defined $title;
		$title =~ s/\s+$//;
		return unless length $title;
		return if $title eq ${*$self}{'lastTitle'};

		${*$self}{'lastTitle'} = $title;

		Slim::Music::Info::setTitle($url, $title);
		Slim::Music::Info::setCurrentTitle($url, $title, $client);
	};

	my $timer;
	$timer = sub {
		${*$self}{'titleTimer'} = Slim::Utils::Timers::setTimer(
			$client, time() + 2, $timer
		);

		$apply->();
	};

	# Apply a known title right away, so the player does not briefly show the
	# stream name when a new pipeline starts.
	$apply->();

	${*$self}{'titleTimer'} = Slim::Utils::Timers::setTimer($client, time() + 2, $timer);
}

sub close {
	my $self = shift;

	if (my $timer = ${*$self}{'titleTimer'}) {
		Slim::Utils::Timers::killSpecific($timer);
		delete ${*$self}{'titleTimer'};
	}

	return $self->SUPER::close(@_);
}

sub canDoAction {
	my ($class, $client, $url, $action) = @_;

	$log->debug("Action=$action for $url");

	return 1;
}

sub canHandleTranscode {
	my ($self, $song) = @_;

	return 1;
}

sub getStreamBitrate {
	my ($self, $maxRate) = @_;

	return Slim::Player::Song::guessBitrateFromFormat(${*$self}{'contentType'}, $maxRate);
}

sub isAudioURL { 1 }

sub scanUrl {
	my ($class, $url, $args) = @_;

	my $song  = $args->{'song'};
	my $track = $song ? $song->track : undef;
	my $cb    = $args->{'cb'} || sub {};

	unless ($class->_valid_url($url)) {
		$log->error("refusing invalid VBAN URL: $url");
		$cb->(undef, 'SCANNER_REMOTE_INVALID_URL');
		return;
	}

	unless ($track) {
		$cb->(undef, 'SCANNER_REMOTE_NO_URL_PROVIDED');
		return;
	}

	# Make sure the content type is known before Song::open negotiates the
	# transcoder, even for a URL that was never played before.
	Slim::Music::Info::setContentType($url, 'vban');

	# A VBAN stream is live audio, never a playlist: no HTTP scan needed. Going
	# through Slim::Utils::Scanner::Remote would try to fetch vban:// over HTTP
	# and never call back, leaving the player in "waitingToPlay".
	$log->debug("scanUrl: $url is a live VBAN stream");

	$cb->($track, undef);
}

sub canDirectStream {
	return 0;
}

sub contentType {
	my $self = shift;

	return ${*$self}{'contentType'};
}

sub getMetadataFor {
	my ($class, $client, $url, $forceCurrent) = @_;

	my $icon = eval { Plugins::VBAN::Plugin->_pluginDataFor('icon') }
		|| 'plugins/VBAN/html/images/vban.png';

	$log->debug("getMetadataFor $url ($icon)");

	return {
		bitrate => 'CD ',
		icon    => $icon,
		cover   => $icon,
		type    => 'VBAN stream',
	};
}

1;
