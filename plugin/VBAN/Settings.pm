package Plugins::VBAN::Settings;

use strict;

use base qw(Slim::Web::Settings);

use Slim::Utils::Prefs;

my $prefs = preferences('plugin.VBAN');

sub name {
	return 'PLUGIN_VBAN';
}

sub page {
	return 'plugins/VBAN/settings/basic.html';
}

sub prefs {
	return ($prefs, qw(host port stream));
}

1;
