from pyln.client import RpcError
from pyln.testing.fixtures import *
from pyln.testing.utils import wait_for

def test_bcli(node_factory, bitcoind, chainparams):
    """
    This tests the bcli plugin, used to gather Bitcoin data from a local
    bitcoind.
    Mostly sanity checks of the interface...
    """

    l1 = node_factory.get_node()
    l2 = node_factory.get_node()

    # We cant stop it dynamically
    with pytest.raises(RpcError):
        l1.rpc.plugin_stop("bcli")

    # Failure case of feerate is tested in test_misc.py
    estimates = l1.rpc.call("estimatefees")
    assert 'feerate_floor' in estimates
    assert [f['blocks'] for f in estimates['feerates']] == [2, 6, 12, 100]

    resp = l1.rpc.call("getchaininfo", {"last_height": 0})
    assert resp["chain"] == chainparams['name']
    for field in ["headercount", "blockcount", "ibd"]:
        assert field in resp

    # We shouldn't get upset if we ask for an unknown-yet block
    resp = l1.rpc.call("getrawblockbyheight", {"height": 500})
    assert resp["blockhash"] is resp["block"] is None
    resp = l1.rpc.call("getrawblockbyheight", {"height": 50})
    assert resp["blockhash"] is not None and resp["blockhash"] is not None
    # Some other bitcoind-failure cases for this call are covered in
    # tests/test_misc.py

    l1.fundwallet(10**5)
    l1.connect(l2)
    fc = l1.rpc.fundchannel(l2.info["id"], 10**4 * 3)
    txo = l1.rpc.call("getutxout", {"txid": fc['txid'], "vout": fc['outnum']})
    assert (Millisatoshi(txo["amount"]) == Millisatoshi(10**4 * 3 * 10**3)
            and txo["script"].startswith("0020"))
    l1.rpc.close(l2.info["id"])
    # When output is spent, it should give us null !
    wait_for(lambda: l1.rpc.call("getutxout", {
        "txid": fc['txid'],
        "vout": fc['outnum']
    })['amount'] is None)

    resp = l1.rpc.call("sendrawtransaction", {"tx": "dummy", "allowhighfees": False})
    assert not resp["success"] and "decode failed" in resp["errmsg"]
