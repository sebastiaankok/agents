I started a new repository to hold all my agents skills so i can install them easily. I also want to develop a cli utility to run opencode containers. the idea is;

1. Have python/Go cli tool that fetches github/gitlab issues. For issues with label ready-for-agent, it creates a container with opencode that starts working on the issue. When a issue is blocked by another issue, it needs to sleep until it no longer is blocked.

2. PR's need to be automatically merged when the pipeline succeeds (optionally) this allows to let "sleeped" jobs to continue whenever possible. important to git pull before working on it.

3. We need to make skills available to the opencode container.

4. Contains can run either locally, or k8s cluster if there is one in context.

5. The tool needs to be ran from the repository directory



