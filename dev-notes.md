# Development notes

Notes on `real-cluster-experiment/main.go`:

* Watching resource state is done in a very basic way, only using
    `event.Reason` only. It is sufficient for now but not robust enough with
    multiple resource types.
* Jobs finishing state is assumed to be COMPLETED_SUCCESSFULLY
* Because using events timestamps results in negative values in the csv,
    timestamps are mesured upon reception of the event in this script resulting
    in a slight delay compared to the actual values. This is not an issue
    considering the (often) very long simulation times.
* Kubernetes won't allow to schedule a pod that would utilize 100% of the
    resources of a node. For example, a pod requiring 6 cpu won't be scheduled
    on a node having 6 allocatable cpu. This was the object of a lot of
    qestioning on why the experiments failed on the scheduler stating that a
    node did not have enough cpu available. Leaving 100m cpu left on the node
    works better.
