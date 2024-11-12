from pyln.client import RpcError
from pyln.testing.fixtures import *

def test_bcli(node_factory, bitcoind, chainparams):
    """
    Based on the test_bcli from Core Lightning
    """
    node = node_factory.get_node(opts={
        "disable-plugin": "bcli",
        "plugin": os.path.join(os.getcwd(), 'trustedcoin'),
    })

    # We cant stop it dynamically
    with pytest.raises(RpcError):
        node.rpc.plugin_stop("bcli")

    # Failure case of feerate is tested in test_misc.py
    estimates = node.rpc.call("estimatefees")
    assert 'feerate_floor' in estimates
    assert [f['blocks'] for f in estimates['feerates']] == [2, 6, 12, 100]

    resp = node.rpc.call("getchaininfo", {"last_height": 0})
    assert resp["chain"] == chainparams['name']
    for field in ["headercount", "blockcount", "ibd"]:
        assert field in resp

    # We shouldn't get upset if we ask for an unknown-yet block
    resp = node.rpc.call("getrawblockbyheight", {"height": 500})
    assert resp["blockhash"] is resp["block"] is None
    resp = node.rpc.call("getrawblockbyheight", {"height": 50})
    assert resp["blockhash"] is not None and resp["blockhash"] is not None
