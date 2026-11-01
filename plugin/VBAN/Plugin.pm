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

use base qw(Slim::Plugin::Base);

use File::Basename qw(dirname);
use File::Spec::Functions qw(catdir catfile);

use Slim::Utils::Log;
use Slim::Utils::Misc;
use Slim::Utils::OSDetect;

# Create the log category before loading the protocol handler.
my $log = Slim::Utils::Log->addLogCategory({
	'category'     => 'plugin.VBAN',
	'defaultLevel' => 'INFO',
	'description'  => getDisplayName(),
});

use Plugins::VBAN::VBAN;

################################
### Plugin Interface ###########
################################

sub initPlugin {
	my $class = shift;

	$class->_setupBinPath();

	$log->info('Initialising ' . $class->_pluginDataFor('version') . ' on ' . $^O);

	return 1;
}

sub shutdownPlugin {
	return;
}

sub getDisplayName {
	return ('PLUGIN_VBAN');
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
