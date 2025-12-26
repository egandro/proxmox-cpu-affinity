package PVE::API2::Nodes::HelloWorld;

use strict;
use warnings;
use PVE::RESTHandler;
use JSON;

use base qw(PVE::RESTHandler);

__PACKAGE__->register_method ({
    name => 'helloworld',
    path => 'helloworld',
    method => 'GET',
    description => "Call the Go backend",
    proxyto => 'node',
    permissions => {
        check => ['perm', '/', [ 'Sys.Audit' ]],
    },
    parameters => {
        additionalProperties => 0,
        properties => {
            node => { type => 'string', format => 'pve-node-id' },
        },
    },
    returns => {
        type => 'object',
        properties => {
            message => { type => 'string' },
        },
    },
    code => sub {
        my ($param) = @_;

        # Path to your compiled Go binary
        my $cmd = '/usr/local/bin/pve-hello-go';

        # Execute the Go binary and capture output
        my $output = `$cmd`;

        # Decode the JSON from Go
        my $json_result = decode_json($output);

        return { message => $json_result->{message} };
    }});

1;
