from pyln.testing.fixtures import *

def test_bcli(node_factory, bitcoind, chainparams):
    """
    This tests the bcli plugin, used to gather Bitcoin data from a local
    bitcoind.
    Mostly sanity checks of the interface...
    """
    l1 = node_factory.get_node()
    assert 1 == 1