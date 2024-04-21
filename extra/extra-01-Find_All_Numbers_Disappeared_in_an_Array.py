class Solution(object):
    def findDisappearedNumbers(self, nums):
        l = len(nums)
        s = set(nums)
        arr = []
        for i in range(1, l+1):
            if i not in s:
                arr.append(i)
        return arr 