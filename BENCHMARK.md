## Prerequisites
1. Install the NVIDIA drivers
2. Select nvidia using `prime-select`
```
sudo apt -y install prime-select
sudo prime-select nvidia
```
3. Reboot
4. Install tools for generating some GPU load
```
sudo apt -y install mesa-utils glmark2
```

## Generate GPU utilization
Run one of the following
```
glxgears
```
or
```
glmark2
```