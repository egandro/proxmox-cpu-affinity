// Wait for the PVE environment to load
Ext.define('PVE.HelloWorldPlugin', {
    override: 'PVE.node.Summary', // We override the Node Summary view

    initComponent: function() {
        var me = this;

        // Call the original initComponent
        me.callParent();

        // Find the top toolbar (tbar)
        var tbar = me.down('tbar');

        if (tbar) {
            tbar.add('-'); // Add a visual separator
            tbar.add({
                text: 'Hello Go',
                iconCls: 'fa fa-bolt', // FontAwesome icon
                handler: function() {
                    // Call the Perl API -> which calls Go
                    Proxmox.Utils.API2Request({
                        url: '/nodes/' + me.pveSelNode.data.node + '/helloworld',
                        method: 'GET',
                        waitMsgTarget: me,
                        success: function(response) {
                            // Update a text or show alert
                            Ext.Msg.alert('Go Backend Says:', response.result.data.message);
                        },
                        failure: function(response) {
                            Ext.Msg.alert('Error', response.htmlStatus);
                        }
                    });
                }
            });
        }
    }
});
