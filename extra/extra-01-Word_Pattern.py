class Solution(object):
    def wordPattern(self, pattern, s):
        arr = s.split()
        if len(pattern) != len(arr):
            return False

        for i in range(len(arr)):
            if pattern.find(pattern[i]) != arr.index(arr[i]):
                return False
        return True